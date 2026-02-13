package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
)

// Bill represents a bill for water usage
type Bill struct {
	ID             string          `json:"id"`
	SubscriberID   string          `json:"subscriber_id"`
	AccountNumber  string          `json:"account_number"`
	BillNumber     string          `json:"bill_number"`
	PeriodStart    time.Time       `json:"period_start"`
	PeriodEnd      time.Time       `json:"period_end"`
	IssuedAt       time.Time       `json:"issued_at"`
	DueDate        time.Time       `json:"due_date"`

	// Usage details
	HotWaterUsage  decimal.Decimal `json:"hot_water_usage"`   // m³
	ColdWaterUsage decimal.Decimal `json:"cold_water_usage"`  // m³

	// Tariffs
	HotWaterTariff decimal.Decimal `json:"hot_water_tariff"`  // per m³
	ColdWaterTariff decimal.Decimal `json:"cold_water_tariff"` // per m³

	// Amounts
	HotWaterAmount decimal.Decimal `json:"hot_water_amount"`
	ColdWaterAmount decimal.Decimal `json:"cold_water_amount"`
	SewageAmount   decimal.Decimal `json:"sewage_amount"`      // wastewater (usually % of total water)
	TotalAmount    decimal.Decimal `json:"total_amount"`

	// Status
	Status         string    `json:"status"` // draft, issued, paid, overdue, cancelled
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	PaymentID      string    `json:"payment_id,omitempty"`

	// Additional charges
	ServiceCharge  decimal.Decimal `json:"service_charge"`
	MaintenanceCharge decimal.Decimal `json:"maintenance_charge"`

	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Tariff represents water pricing
type Tariff struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	HotWaterPrice  decimal.Decimal `json:"hot_water_price"`   // per m³
	ColdWaterPrice decimal.Decimal `json:"cold_water_price"`  // per m³
	SewagePrice    decimal.Decimal `json:"sewage_price"`      // per m³
	ServiceCharge  decimal.Decimal `json:"service_charge"`    // fixed monthly
	ValidFrom      time.Time       `json:"valid_from"`
	ValidTo        *time.Time      `json:"valid_to,omitempty"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
}

// Payment represents a payment transaction
type Payment struct {
	ID            string          `json:"id"`
	BillID        string          `json:"bill_id"`
	SubscriberID  string          `json:"subscriber_id"`
	Amount        decimal.Decimal `json:"amount"`
	Method        string          `json:"method"` // card, cash, bank_transfer
	Status        string          `json:"status"` // pending, completed, failed, refunded
	ExternalID    string          `json:"external_id,omitempty"` // payment gateway ID
	CreatedAt     time.Time       `json:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	FailedAt      *time.Time      `json:"failed_at,omitempty"`
	ErrorMessage  string          `json:"error_message,omitempty"`
}

var (
	billsDB       = make(map[string]Bill)
	tariffsDB     = make(map[string]Tariff)
	paymentsDB    = make(map[string]Payment)
	dbMutex       sync.RWMutex
	rabbitMQURL   = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize default tariff
	defaultTariff := Tariff{
		ID:             "tariff_default",
		Name:           "Базовый тариф",
		HotWaterPrice:  decimal.NewFromFloat(150.50),  // руб за м³
		ColdWaterPrice: decimal.NewFromFloat(45.20),   // руб за м³
		SewagePrice:    decimal.NewFromFloat(30.00),   // руб за м³
		ServiceCharge:  decimal.NewFromFloat(350.00),  // руб фиксированная
		ValidFrom:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	tariffsDB[defaultTariff.ID] = defaultTariff
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Bill endpoints
	api.HandleFunc("/bills", getBills).Methods("GET")
	api.HandleFunc("/bills/{id}", getBill).Methods("GET")
	api.HandleFunc("/bills", createBill).Methods("POST")
	api.HandleFunc("/bills/{id}", updateBill).Methods("PUT")
	api.HandleFunc("/bills/{id}/cancel", cancelBill).Methods("POST")
	api.HandleFunc("/bills/subscriber/{subscriberId}", getSubscriberBills).Methods("GET")
	api.HandleFunc("/bills/account/{accountNumber}", getAccountBills).Methods("GET")
	api.HandleFunc("/bills/{id}/recalculate", recalculateBill).Methods("POST")

	// Tariff endpoints
	api.HandleFunc("/tariffs", getTariffs).Methods("GET")
	api.HandleFunc("/tariffs/{id}", getTariff).Methods("GET")
	api.HandleFunc("/tariffs", createTariff).Methods("POST")
	api.HandleFunc("/tariffs/{id}", updateTariff).Methods("PUT")
	api.HandleFunc("/tariffs/{id}", deleteTariff).Methods("DELETE")
	api.HandleFunc("/tariffs/active", getActiveTariff).Methods("GET")

	// Payment endpoints (for integration with payments service)
	api.HandleFunc("/payments", getPayments).Methods("GET")
	api.HandleFunc("/payments/{id}", getPayment).Methods("GET")
	api.HandleFunc("/bills/{billId}/payments", getBillPayments).Methods("GET")
	api.HandleFunc("/payments/{id}/confirm", confirmPayment).Methods("POST")

	// Report endpoints
	api.HandleFunc("/reports/debtors", getDebtorsReport).Methods("GET")
	api.HandleFunc("/reports/monthly", getMonthlyReport).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	// Start RabbitMQ consumer for meter readings
	go startRabbitMQConsumer()

	// Start bill generation scheduler
	go startBillScheduler()

	srv := &http.Server{
		Addr:         ":8084",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Billing Service starting on port 8084")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// Handlers
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "billing",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Bill handlers
func getBills(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	bills := make([]Bill, 0, len(billsDB))
	for _, b := range billsDB {
		bills = append(bills, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bills)
}

func getBill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	bill, exists := billsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Bill not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bill)
}

func createBill(w http.ResponseWriter, r *http.Request) {
	var bill Bill
	if err := json.NewDecoder(r.Body).Decode(&bill); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bill.ID = fmt.Sprintf("bill_%d", time.Now().UnixNano())
	bill.BillNumber = generateBillNumber(bill.SubscriberID, bill.PeriodStart)
	bill.Status = "draft"
	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	// Calculate amounts
	calculateBillAmounts(&bill)

	dbMutex.Lock()
	billsDB[bill.ID] = bill
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bill)
}

func updateBill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var bill Bill
	if err := json.NewDecoder(r.Body).Decode(&bill); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := billsDB[id]; !exists {
		http.Error(w, "Bill not found", http.StatusNotFound)
		return
	}

	bill.ID = id
	bill.UpdatedAt = time.Now()
	calculateBillAmounts(&bill)
	billsDB[id] = bill

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bill)
}

func cancelBill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	bill, exists := billsDB[id]
	if !exists {
		http.Error(w, "Bill not found", http.StatusNotFound)
		return
	}

	if bill.Status == "paid" {
		http.Error(w, "Cannot cancel paid bill", http.StatusBadRequest)
		return
	}

	bill.Status = "cancelled"
	bill.UpdatedAt = time.Now()
	billsDB[id] = bill

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bill)
}

func recalculateBill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	bill, exists := billsDB[id]
	if !exists {
		http.Error(w, "Bill not found", http.StatusNotFound)
		return
	}

	if bill.Status == "paid" {
		http.Error(w, "Cannot recalculate paid bill", http.StatusBadRequest)
		return
	}

	calculateBillAmounts(&bill)
	bill.UpdatedAt = time.Now()
	billsDB[id] = bill

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bill)
}

func getSubscriberBills(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriberID := vars["subscriberId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	subscriberBills := make([]Bill, 0)
	for _, b := range billsDB {
		if b.SubscriberID == subscriberID {
			subscriberBills = append(subscriberBills, b)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriberBills)
}

func getAccountBills(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountNumber := vars["accountNumber"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	accountBills := make([]Bill, 0)
	for _, b := range billsDB {
		if b.AccountNumber == accountNumber {
			accountBills = append(accountBills, b)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accountBills)
}

// Tariff handlers
func getTariffs(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	tariffs := make([]Tariff, 0, len(tariffsDB))
	for _, t := range tariffsDB {
		tariffs = append(tariffs, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tariffs)
}

func getTariff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	tariff, exists := tariffsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Tariff not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tariff)
}

func createTariff(w http.ResponseWriter, r *http.Request) {
	var tariff Tariff
	if err := json.NewDecoder(r.Body).Decode(&tariff); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tariff.ID = fmt.Sprintf("tariff_%d", time.Now().UnixNano())
	tariff.CreatedAt = time.Now()

	dbMutex.Lock()
	tariffsDB[tariff.ID] = tariff
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tariff)
}

func updateTariff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var tariff Tariff
	if err := json.NewDecoder(r.Body).Decode(&tariff); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := tariffsDB[id]; !exists {
		http.Error(w, "Tariff not found", http.StatusNotFound)
		return
	}

	tariff.ID = id
	tariffsDB[id] = tariff

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tariff)
}

func deleteTariff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := tariffsDB[id]; !exists {
		http.Error(w, "Tariff not found", http.StatusNotFound)
		return
	}

	delete(tariffsDB, id)
	w.WriteHeader(http.StatusNoContent)
}

func getActiveTariff(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	now := time.Now()
	for _, t := range tariffsDB {
		if t.IsActive && now.After(t.ValidFrom) && (t.ValidTo == nil || now.Before(*t.ValidTo)) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)
			return
		}
	}

	http.Error(w, "No active tariff found", http.StatusNotFound)
}

// Payment handlers
func getPayments(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	payments := make([]Payment, 0, len(paymentsDB))
	for _, p := range paymentsDB {
		payments = append(payments, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	payment, exists := paymentsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func getBillPayments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	billID := vars["billId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	billPayments := make([]Payment, 0)
	for _, p := range paymentsDB {
		if p.BillID == billID {
			billPayments = append(billPayments, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(billPayments)
}

func confirmPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data struct {
		BillID   string `json:"bill_id"`
		Amount   string `json:"amount"`
		ExternalID string `json:"external_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	amount, err := decimal.NewFromString(data.Amount)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Create payment record
	payment := Payment{
		ID:        id,
		BillID:    data.BillID,
		Amount:    amount,
		Status:    "completed",
		ExternalID: data.ExternalID,
		CreatedAt: time.Now(),
	}
	now := time.Now()
	payment.CompletedAt = &now
	paymentsDB[id] = payment

	// Update bill status
	if bill, exists := billsDB[data.BillID]; exists {
		bill.Status = "paid"
		bill.PaymentID = id
		bill.PaidAt = &now
		bill.UpdatedAt = now
		billsDB[data.BillID] = bill

		// Send notification via RabbitMQ
		go sendPaymentNotification(bill)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

// Report handlers
func getDebtorsReport(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	now := time.Now()
	debtors := make([]Bill, 0)

	for _, b := range billsDB {
		if (b.Status == "issued" || b.Status == "overdue") && b.DueDate.Before(now) {
			if b.Status == "issued" {
				b.Status = "overdue"
			}
			debtors = append(debtors, b)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_debtors": len(debtors),
		"total_amount": calculateTotalDebt(debtors),
		"bills": debtors,
	})
}

func getMonthlyReport(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		monthStr = time.Now().Format("2006-01")
	}

	parsedMonth, _ := time.Parse("2006-01", monthStr)

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	monthlyBills := make([]Bill, 0)
	totalIssued := decimal.Zero
	totalPaid := decimal.Zero

	for _, b := range billsDB {
		if b.PeriodStart.Year() == parsedMonth.Year() &&
		   b.PeriodStart.Month() == parsedMonth.Month() {
			monthlyBills = append(monthlyBills, b)
			totalIssued = totalIssued.Add(b.TotalAmount)
			if b.Status == "paid" {
				totalPaid = totalPaid.Add(b.TotalAmount)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"month": monthStr,
		"total_bills": len(monthlyBills),
		"total_issued": totalIssued.String(),
		"total_paid": totalPaid.String(),
		"bills": monthlyBills,
	})
}

// Helper functions
func calculateBillAmounts(bill *Bill) {
	hotAmount := bill.HotWaterUsage.Mul(bill.HotWaterTariff)
	coldAmount := bill.ColdWaterUsage.Mul(bill.ColdWaterTariff)

	totalWater := bill.HotWaterUsage.Add(bill.ColdWaterUsage)
	sewageAmount := totalWater.Mul(decimal.NewFromFloat(30.00)) // Default sewage price

	bill.HotWaterAmount = hotAmount
	bill.ColdWaterAmount = coldAmount
	bill.SewageAmount = sewageAmount

	total := hotAmount.Add(coldAmount).Add(sewageAmount)
	total = total.Add(bill.ServiceCharge)
	total = total.Add(bill.MaintenanceCharge)

	bill.TotalAmount = total
}

func generateBillNumber(subscriberID string, period time.Time) string {
	return fmt.Sprintf("В-%s-%s", subscriberID, period.Format("200601"))
}

func calculateTotalDebt(bills []Bill) string {
	total := decimal.Zero
	for _, b := range bills {
		total = total.Add(b.TotalAmount)
	}
	return total.String()
}

// RabbitMQ consumer for meter readings
func startRabbitMQConsumer() {
	for {
		conn, err := amqp.Dial(rabbitMQURL)
		if err != nil {
			log.Printf("Failed to connect to RabbitMQ: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Failed to open channel: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Declare queue for readings
		q, err := ch.QueueDeclare(
			"billing_readings_queue",
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			log.Printf("Failed to declare queue: %v", err)
			ch.Close()
			conn.Close()
			continue
		}

		msgs, err := ch.Consume(
			q.Name,
			"",
			false, // auto-ack
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Failed to register consumer: %v", err)
			ch.Close()
			conn.Close()
			continue
		}

		log.Println("Billing RabbitMQ consumer started")

		for d := range msgs {
			processReadingMessage(&d)
			d.Ack(false)
		}

		ch.Close()
		conn.Close()
	}
}

func processReadingMessage(msg *amqp.Delivery) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Body, &data); err != nil {
		log.Printf("Failed to unmarshal reading: %v", err)
		return
	}

	log.Printf("Received reading for billing: %+v", data)
	// In real implementation, this would calculate and generate bills
}

// Bill generation scheduler
func startBillScheduler() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		generateMonthlyBills()
	}
}

func generateMonthlyBills() {
	log.Println("Generating monthly bills...")
	// In real implementation, this would:
	// 1. Get all subscribers
	// 2. Get their meter readings for the period
	// 3. Calculate amounts based on tariffs
	// 4. Create bill records
	// 5. Send notifications
}

func sendPaymentNotification(bill Bill) {
	// In real implementation, publish to RabbitMQ for notifications service
	log.Printf("Payment notification for bill %s", bill.ID)
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s completed in %v", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

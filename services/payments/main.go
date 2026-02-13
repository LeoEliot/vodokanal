package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// Payment represents a payment transaction
type Payment struct {
	ID             string          `json:"id"`
	BillID         string          `json:"bill_id"`
	SubscriberID   string          `json:"subscriber_id"`
	AccountNumber  string          `json:"account_number"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"` // RUB
	Method         string          `json:"method"`   // card, yookassa, stripe, sbp
	Status         string          `json:"status"`   // pending, processing, completed, failed, refunded, cancelled

	// Payment gateway details
	Gateway        string          `json:"gateway"`         // yookassa, stripe, etc.
	GatewayTxID    string          `json:"gateway_tx_id"`   // transaction ID at gateway
	GatewayStatus  string          `json:"gateway_status"`

	// Callback/Redirect URLs
	SuccessURL     string          `json:"success_url,omitempty"`
	FailURL        string          `json:"fail_url,omitempty"`
	ReturnURL      string          `json:"return_url,omitempty"`

	// Metadata
	Description    string          `json:"description"`
	Metadata       map[string]string `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	FailedAt       *time.Time      `json:"failed_at,omitempty"`

	// Refund details
	RefundedAmount decimal.Decimal `json:"refunded_amount"`
	RefundReason   string          `json:"refund_reason,omitempty"`
	RefundedAt     *time.Time      `json:"refunded_at,omitempty"`
}

// PaymentRequest represents a request to create a payment
type PaymentRequest struct {
	BillID        string          `json:"bill_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	Method        string          `json:"method"`
	SuccessURL    string          `json:"success_url,omitempty"`
	FailURL       string          `json:"fail_url,omitempty"`
	Description   string          `json:"description,omitempty"`
}

// PaymentResponse represents the response when creating a payment
type PaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	PaymentURL    string `json:"payment_url,omitempty"`     // URL for user to complete payment
	QRCodeURL     string `json:"qr_code_url,omitempty"`     // For SBP payments
	ExpiresAt     time.Time `json:"expires_at"`
}

// Refund represents a refund transaction
type Refund struct {
	ID             string          `json:"id"`
	PaymentID      string          `json:"payment_id"`
	Amount         decimal.Decimal `json:"amount"`
	Reason         string          `json:"reason"`
	Status         string          `json:"status"`
	GatewayRefundID string         `json:"gateway_refund_id"`
	CreatedAt      time.Time       `json:"created_at"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
}

// WebhookEvent represents an incoming webhook from payment gateway
type WebhookEvent struct {
	Gateway string          `json:"gateway"`
	Event   string          `json:"event"`
	Data    json.RawMessage `json:"data"`
}

var (
	paymentsDB    = make(map[string]Payment)
	refundsDB     = make(map[string]Refund)
	dbMutex       sync.RWMutex
	rabbitMQURL   = os.Getenv("RABBITMQ_URL")
	yookassaShopID = os.Getenv("YOOKASSA_SHOP_ID")
	yookassaSecretKey = os.Getenv("YOOKASSA_SECRET_KEY")
	stripeSecretKey   = os.Getenv("STRIPE_SECRET_KEY")
	webhookSecret     = os.Getenv("WEBHOOK_SECRET")
	billingServiceURL = os.Getenv("BILLING_SERVICE_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}
	if billingServiceURL == "" {
		billingServiceURL = "http://billing:8084"
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Payment endpoints
	api.HandleFunc("/payments", getPayments).Methods("GET")
	api.HandleFunc("/payments/{id}", getPayment).Methods("GET")
	api.HandleFunc("/payments", createPayment).Methods("POST")
	api.HandleFunc("/payments/{id}/cancel", cancelPayment).Methods("POST")
	api.HandleFunc("/payments/{id}/status", getPaymentStatus).Methods("GET")
	api.HandleFunc("/bills/{billId}/payments", getBillPayments).Methods("GET")
	api.HandleFunc("/subscribers/{subscriberId}/payments", getSubscriberPayments).Methods("GET")

	// Refund endpoints
	api.HandleFunc("/payments/{id}/refund", createRefund).Methods("POST")
	api.HandleFunc("/refunds", getRefunds).Methods("GET")
	api.HandleFunc("/refunds/{id}", getRefund).Methods("GET")

	// Webhook endpoints (no auth required, verified via signature)
	r.HandleFunc("/webhooks/yookassa", handleYookassaWebhook).Methods("POST")
	r.HandleFunc("/webhooks/stripe", handleStripeWebhook).Methods("POST")
	r.HandleFunc("/webhooks/sbp", handleSBPWebhook).Methods("POST")

	// Payment methods info
	api.HandleFunc("/payment-methods", getPaymentMethods).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	// Start webhook processors
	go startWebhookProcessor()

	srv := &http.Server{
		Addr:         ":8085",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Payments Service starting on port 8085")
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
		"service":   "payments",
		"timestamp": time.Now().Format(time.RFC3339),
	})
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

func createPayment(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Amount.IsZero() || req.Amount.IsNegative() {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	if req.Currency == "" {
		req.Currency = "RUB"
	}

	if req.Method == "" {
		req.Method = "card"
	}

	// Create payment
	payment := Payment{
		ID:            fmt.Sprintf("pay_%d", time.Now().UnixNano()),
		BillID:        req.BillID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Method:        req.Method,
		Status:        "pending",
		SuccessURL:    req.SuccessURL,
		FailURL:       req.FailURL,
		Description:   req.Description,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Select gateway based on method
	switch req.Method {
	case "yookassa", "card":
		payment.Gateway = "yookassa"
	case "stripe":
		payment.Gateway = "stripe"
	case "sbp":
		payment.Gateway = "sbp"
	default:
		payment.Gateway = "yookassa"
	}

	// Initiate payment with gateway
	response, err := initiateGatewayPayment(&payment)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initiate payment: %v", err), http.StatusInternalServerError)
		return
	}

	dbMutex.Lock()
	paymentsDB[payment.ID] = payment
	dbMutex.Unlock()

	// Publish payment created event
	go publishPaymentEvent(payment, "created")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func cancelPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	payment, exists := paymentsDB[id]
	if !exists {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	if payment.Status != "pending" && payment.Status != "processing" {
		http.Error(w, "Cannot cancel payment in current status", http.StatusBadRequest)
		return
	}

	payment.Status = "cancelled"
	payment.UpdatedAt = time.Now()
	paymentsDB[id] = payment

	// Publish cancelled event
	go publishPaymentEvent(payment, "cancelled")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func getPaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	payment, exists := paymentsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	// Check status with gateway if not final
	if payment.Status == "pending" || payment.Status == "processing" {
		status := checkGatewayStatus(&payment)
		if status != payment.Status {
			dbMutex.Lock()
			payment.Status = status
			payment.UpdatedAt = time.Now()
			paymentsDB[id] = payment
			dbMutex.Unlock()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"payment_id": payment.ID,
		"status":     payment.Status,
		"gateway_status": payment.GatewayStatus,
	})
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

func getSubscriberPayments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriberID := vars["subscriberId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	subscriberPayments := make([]Payment, 0)
	for _, p := range paymentsDB {
		if p.SubscriberID == subscriberID {
			subscriberPayments = append(subscriberPayments, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriberPayments)
}

// Refund handlers
func createRefund(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Amount string `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	payment, exists := paymentsDB[id]
	if !exists {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	if payment.Status != "completed" {
		http.Error(w, "Can only refund completed payments", http.StatusBadRequest)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.IsZero() || amount.IsNegative() {
		http.Error(w, "Invalid refund amount", http.StatusBadRequest)
		return
	}

	if amount.GreaterThan(payment.Amount) {
		http.Error(w, "Refund amount cannot exceed payment amount", http.StatusBadRequest)
		return
	}

	// Create refund
	refund := Refund{
		ID:        fmt.Sprintf("ref_%d", time.Now().UnixNano()),
		PaymentID: id,
		Amount:    amount,
		Reason:    req.Reason,
		Status:    "processing",
		CreatedAt: time.Now(),
	}

	refundsDB[refund.ID] = refund

	// Process refund with gateway
	go processRefund(refund, payment)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(refund)
}

func getRefunds(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	refunds := make([]Refund, 0, len(refundsDB))
	for _, r := range refundsDB {
		refunds = append(refunds, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refunds)
}

func getRefund(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	refund, exists := refundsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Refund not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refund)
}

func getPaymentMethods(w http.ResponseWriter, r *http.Request) {
	methods := []map[string]interface{}{
		{
			"id":          "card",
			"name":        "Банковская карта",
			"description": "Visa, MasterCard, МИР",
			"icon":        "credit-card",
			"enabled":     true,
		},
		{
			"id":          "sbp",
			"name":        "СБП (Система быстрых платежей)",
			"description": "Оплата по QR-коду через мобильный банк",
			"icon":        "qr",
			"enabled":     true,
		},
		{
			"id":          "yoomoney",
			"name":        "ЮMoney",
			"description": "Оплата из кошелька ЮMoney",
			"icon":        "wallet",
			"enabled":     true,
		},
		{
			"id":          "sberpay",
			"name":        "SberPay",
			"description": "Оплата через SberPay",
			"icon":        "smartphone",
			"enabled":     true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(methods)
}

// Webhook handlers
func handleYookassaWebhook(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode Yookassa webhook: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Verify signature (in production)
	// signature := r.Header.Get("Content-Signature")
	// if !verifyYookassaSignature(signature, event) {
	//     http.Error(w, "Invalid signature", http.StatusUnauthorized)
	//     return
	// }

	eventType, _ := event["event"].(string)
	object, _ := event["object"].(map[string]interface{})

	log.Printf("Yookassa webhook: %s", eventType)

	go processYookassaEvent(eventType, object)

	w.WriteHeader(http.StatusOK)
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode Stripe webhook: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Verify signature
	sig := r.Header.Get("Stripe-Signature")
	if !verifyStripeSignature(sig, r.Body, webhookSecret) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	eventType, _ := event["type"].(string)
	log.Printf("Stripe webhook: %s", eventType)

	w.WriteHeader(http.StatusOK)
}

func handleSBPWebhook(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode SBP webhook: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	log.Printf("SBP webhook received")
	w.WriteHeader(http.StatusOK)
}

// Helper functions
func initiateGatewayPayment(payment *Payment) (*PaymentResponse, error) {
	response := &PaymentResponse{
		PaymentID: payment.ID,
		Amount:    payment.Amount.String(),
		Currency:  payment.Currency,
		Status:    "pending",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// In real implementation, call actual gateway APIs
	switch payment.Gateway {
	case "yookassa":
		// Mock Yookassa payment
		payment.GatewayTxID = fmt.Sprintf("yo_%d", time.Now().UnixNano())
		response.PaymentURL = fmt.Sprintf("https://yookassa.app/pay/%s", payment.GatewayTxID)
	case "stripe":
		payment.GatewayTxID = fmt.Sprintf("pi_%d", time.Now().UnixNano())
		response.PaymentURL = fmt.Sprintf("https://checkout.stripe.com/pay/%s", payment.GatewayTxID)
	case "sbp":
		payment.GatewayTxID = fmt.Sprintf("sbp_%d", time.Now().UnixNano())
		response.QRCodeURL = fmt.Sprintf("https://qr.nspk.ru/%s", payment.GatewayTxID)
	}

	return response, nil
}

func checkGatewayStatus(payment *Payment) string {
	// In real implementation, check actual gateway status
	// For now, return current status
	return payment.Status
}

func processYookassaEvent(eventType string, object map[string]interface{}) {
	paymentID, _ := object["merchant_customer_id"].(string)
	if paymentID == "" {
		// Try to find by metadata
		if metadata, ok := object["metadata"].(map[string]interface{}); ok {
			if id, ok := metadata["payment_id"].(string); ok {
				paymentID = id
			}
		}
	}

	if paymentID == "" {
		log.Printf("Cannot find payment ID in Yookassa event")
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	payment, exists := paymentsDB[paymentID]
	if !exists {
		log.Printf("Payment not found: %s", paymentID)
		return
	}

	status, _ := object["status"].(string)
	now := time.Now()

	switch status {
	case "succeeded":
		payment.Status = "completed"
		payment.GatewayStatus = "succeeded"
		payment.CompletedAt = &now
		// Notify billing service
		go notifyBillingService(payment)
	case "canceled":
		payment.Status = "cancelled"
		payment.GatewayStatus = "canceled"
	case "waiting_for_capture":
		payment.Status = "processing"
		payment.GatewayStatus = "waiting_for_capture"
	}

	payment.UpdatedAt = now
	paymentsDB[paymentID] = payment

	// Publish event
	go publishPaymentEvent(payment, "updated")
}

func processRefund(refund Refund, payment Payment) {
	// In real implementation, call actual gateway refund API
	time.Sleep(2 * time.Second) // Simulate processing

	dbMutex.Lock()
	defer dbMutex.Unlock()

	now := time.Now()
	refund.Status = "completed"
	refund.ProcessedAt = &now
	refundsDB[refund.ID] = refund

	payment.RefundedAmount = payment.RefundedAmount.Add(refund.Amount)
	payment.RefundReason = refund.Reason
	payment.RefundedAt = &now
	paymentsDB[payment.ID] = payment

	go publishPaymentEvent(payment, "refunded")
}

func notifyBillingService(payment Payment) {
	// In real implementation, make HTTP call to billing service
	log.Printf("Notifying billing service about payment %s", payment.ID)
}

func publishPaymentEvent(payment Payment, eventType string) {
	// Publish to RabbitMQ for notifications service
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return
	}
	defer ch.Close()

	// Declare exchange
	err = ch.ExchangeDeclare(
		"payments",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		return
	}

	event := map[string]interface{}{
		"event_type":   eventType,
		"payment_id":   payment.ID,
		"bill_id":      payment.BillID,
		"subscriber_id": payment.SubscriberID,
		"amount":       payment.Amount.String(),
		"status":       payment.Status,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(event)
	err = ch.PublishWithContext(
		context.Background(),
		"payments",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish event: %v", err)
	}
}

func startWebhookProcessor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		checkPendingPayments()
	}
}

func checkPendingPayments() {
	dbMutex.RLock()
	pendingPayments := make([]Payment, 0)
	for _, p := range paymentsDB {
		if p.Status == "pending" || p.Status == "processing" {
			pendingPayments = append(pendingPayments, p)
		}
	}
	dbMutex.RUnlock()

	for _, p := range pendingPayments {
		status := checkGatewayStatus(&p)
		if status != p.Status {
			dbMutex.Lock()
			p.Status = status
			p.UpdatedAt = time.Now()
			paymentsDB[p.ID] = p
			dbMutex.Unlock()
		}
	}
}

func verifyYookassaSignature(signature string, event map[string]interface{}) bool {
	// In real implementation, verify Yookassa webhook signature
	return true
}

func verifyStripeSignature(signature string, body io.ReadCloser, secret string) bool {
	// In real implementation, verify Stripe webhook signature
	return true
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

// HMAC SHA256 helper
func hmacSHA256(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

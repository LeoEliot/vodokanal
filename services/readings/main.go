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

// Reading represents a water meter reading
type Reading struct {
	ID              string          `json:"id"`
	SubscriberID    string          `json:"subscriber_id"`
	AccountNumber   string          `json:"account_number"`
	CounterID       string          `json:"counter_id"`

	// Reading values
	PreviousValue   decimal.Decimal `json:"previous_value"`
	CurrentValue    decimal.Decimal `json:"current_value"`
	Consumption     decimal.Decimal `json:"consumption"` // Calculated difference

	// Reading metadata
	ReadingDate     time.Time       `json:"reading_date"`
	SubmissionDate  time.Time       `json:"submission_date"`

	// Source and verification
	Source          string          `json:"source"`          // user, employee, automatic
	SubmittedBy     string          `json:"submitted_by"`    // user ID
	VerifiedBy      string          `json:"verified_by,omitempty"`
	VerifiedAt      *time.Time      `json:"verified_at,omitempty"`
	IsVerified      bool            `json:"is_verified"`

	// Status
	Status          string          `json:"status"`          // draft, submitted, verified, rejected

	// Attachments (photo of meter)
	PhotoURL        string          `json:"photo_url,omitempty"`
	Notes           string          `json:"notes,omitempty"`

	// Timestamps
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CreateReadingRequest represents a request to create a reading
type CreateReadingRequest struct {
	SubscriberID  string          `json:"subscriber_id"`
	CounterID     string          `json:"counter_id"`
	CurrentValue  decimal.Decimal `json:"current_value"`
	ReadingDate   time.Time       `json:"reading_date"`
	PhotoURL      string          `json:"photo_url,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// UpdateReadingRequest represents a request to update a reading
type UpdateReadingRequest struct {
	CurrentValue  decimal.Decimal `json:"current_value"`
	ReadingDate   time.Time       `json:"reading_date"`
	PhotoURL      string          `json:"photo_url,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// VerifyReadingRequest represents a request to verify a reading
type VerifyReadingRequest struct {
	Approved   bool    `json:"approved"`
	Notes      string  `json:"notes,omitempty"`
	VerifiedBy string  `json:"verified_by"`
}

var (
	readingsDB   = make(map[string]Reading)
	dbMutex      sync.RWMutex
	rabbitMQURL  = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize with sample readings
	now := time.Now()
	readingsDB["reading_1"] = Reading{
		ID:             "reading_1",
		SubscriberID:   "sub_test",
		AccountNumber:  "000123456",
		CounterID:      "counter_hot_1",
		PreviousValue:  decimal.NewFromInt(1000),
		CurrentValue:   decimal.NewFromInt(1050),
		Consumption:    decimal.NewFromInt(50),
		ReadingDate:    now.Add(-30 * 24 * time.Hour),
		SubmissionDate: now.Add(-30 * 24 * time.Hour),
		Source:         "user",
		SubmittedBy:    "user_test",
		IsVerified:     true,
		Status:         "verified",
		CreatedAt:      now.Add(-30 * 24 * time.Hour),
		UpdatedAt:      now.Add(-30 * 24 * time.Hour),
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Reading endpoints
	api.HandleFunc("/readings", getReadings).Methods("GET")
	api.HandleFunc("/readings/{id}", getReading).Methods("GET")
	api.HandleFunc("/readings", createReading).Methods("POST")
	api.HandleFunc("/readings/{id}", updateReading).Methods("PUT")
	api.HandleFunc("/readings/{id}", deleteReading).Methods("DELETE")
	api.HandleFunc("/readings/{id}/verify", verifyReading).Methods("POST")
	api.HandleFunc("/subscribers/{subscriberId}/readings", getSubscriberReadings).Methods("GET")
	api.HandleFunc("/counters/{counterId}/readings", getCounterReadings).Methods("GET")
	api.HandleFunc("/readings/submit", submitReading).Methods("POST")

	// Report endpoints
	api.HandleFunc("/readings/report/monthly", getMonthlyReadingsReport).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	srv := &http.Server{
		Addr:         ":8083",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Readings Service starting on port 8083")
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
		"service":   "readings",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Reading handlers
func getReadings(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	// Parse query parameters
	subscriberID := r.URL.Query().Get("subscriber_id")
	counterID := r.URL.Query().Get("counter_id")
	status := r.URL.Query().Get("status")
	verified := r.URL.Query().Get("verified")

	readings := make([]Reading, 0, len(readingsDB))
	for _, reading := range readingsDB {
		// Apply filters
		if subscriberID != "" && reading.SubscriberID != subscriberID {
			continue
		}
		if counterID != "" && reading.CounterID != counterID {
			continue
		}
		if status != "" && reading.Status != status {
			continue
		}
		if verified == "true" && !reading.IsVerified {
			continue
		}
		if verified == "false" && reading.IsVerified {
			continue
		}
		readings = append(readings, reading)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}

func getReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	reading, exists := readingsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Reading not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reading)
}

func createReading(w http.ResponseWriter, r *http.Request) {
	var req CreateReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SubscriberID == "" || req.CounterID == "" {
		http.Error(w, "Subscriber ID and Counter ID are required", http.StatusBadRequest)
		return
	}

	if req.CurrentValue.IsNegative() {
		http.Error(w, "Current value cannot be negative", http.StatusBadRequest)
		return
	}

	// Get previous reading for this counter
	dbMutex.RLock()
	var previousReading *Reading
	for _, rd := range readingsDB {
		if rd.CounterID == req.CounterID && rd.Status == "verified" {
			if previousReading == nil || rd.ReadingDate.After(previousReading.ReadingDate) {
				rd := rd
				previousReading = &rd
			}
		}
	}

	// Get account number (in real implementation, fetch from subscribers service)
	accountNumber := "000000000" // Default
	for _, rd := range readingsDB {
		if rd.SubscriberID == req.SubscriberID {
			accountNumber = rd.AccountNumber
			break
		}
	}
	dbMutex.RUnlock()

	// Calculate consumption
	var previousValue, consumption decimal.Decimal
	if previousReading != nil {
		previousValue = previousReading.CurrentValue
		consumption = req.CurrentValue.Sub(previousValue)
		if consumption.IsNegative() {
			consumption = decimal.Zero
		}
	} else {
		consumption = req.CurrentValue
	}

	dbMutex.Lock()
	id := fmt.Sprintf("reading_%d", time.Now().UnixNano())
	reading := Reading{
		ID:             id,
		SubscriberID:   req.SubscriberID,
		AccountNumber:  accountNumber,
		CounterID:      req.CounterID,
		PreviousValue:  previousValue,
		CurrentValue:   req.CurrentValue,
		Consumption:    consumption,
		ReadingDate:    req.ReadingDate,
		SubmissionDate: time.Now(),
		Source:         "user",
		IsVerified:     false,
		Status:         "submitted",
		PhotoURL:       req.PhotoURL,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	readingsDB[id] = reading
	dbMutex.Unlock()

	// Publish to RabbitMQ for billing service
	go publishReadingEvent(reading, "submitted")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reading)
}

func submitReading(w http.ResponseWriter, r *http.Request) {
	// Alias for createReading with explicit submission
	createReading(w, r)
}

func updateReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	reading, exists := readingsDB[id]
	if !exists {
		http.Error(w, "Reading not found", http.StatusNotFound)
		return
	}

	// Can only update unverified readings
	if reading.IsVerified {
		http.Error(w, "Cannot update verified reading", http.StatusBadRequest)
		return
	}

	// Update fields
	updated := false
	if !req.CurrentValue.IsZero() {
		reading.CurrentValue = req.CurrentValue
		consumption := req.CurrentValue.Sub(reading.PreviousValue)
		if consumption.IsNegative() {
			consumption = decimal.Zero
		}
		reading.Consumption = consumption
		updated = true
	}
	if !req.ReadingDate.IsZero() {
		reading.ReadingDate = req.ReadingDate
		updated = true
	}
	if req.PhotoURL != "" {
		reading.PhotoURL = req.PhotoURL
		updated = true
	}
	if req.Notes != "" {
		reading.Notes = req.Notes
		updated = true
	}

	if updated {
		reading.Status = "submitted"
		reading.UpdatedAt = time.Now()
		readingsDB[id] = reading
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reading)
}

func deleteReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	reading, exists := readingsDB[id]
	if !exists {
		http.Error(w, "Reading not found", http.StatusNotFound)
		return
	}

	// Can only delete unverified readings
	if reading.IsVerified {
		http.Error(w, "Cannot delete verified reading", http.StatusBadRequest)
		return
	}

	delete(readingsDB, id)
	w.WriteHeader(http.StatusNoContent)
}

func verifyReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req VerifyReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	reading, exists := readingsDB[id]
	if !exists {
		http.Error(w, "Reading not found", http.StatusNotFound)
		return
	}

	now := time.Now()
	reading.VerifiedBy = req.VerifiedBy
	reading.VerifiedAt = &now
	reading.IsVerified = true

	if req.Approved {
		reading.Status = "verified"
	} else {
		reading.Status = "rejected"
		reading.Notes = req.Notes
	}
	reading.UpdatedAt = now
	readingsDB[id] = reading

	// Publish to RabbitMQ
	eventType := "verified"
	if !req.Approved {
		eventType = "rejected"
	}
	go publishReadingEvent(reading, eventType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reading)
}

func getSubscriberReadings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriberID := vars["subscriberId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	readings := make([]Reading, 0)
	for _, reading := range readingsDB {
		if reading.SubscriberID == subscriberID {
			readings = append(readings, reading)
		}
	}

	// Sort by reading date descending
	// In real implementation, use proper sorting

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}

func getCounterReadings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	counterID := vars["counterId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	readings := make([]Reading, 0)
	for _, reading := range readingsDB {
		if reading.CounterID == counterID {
			readings = append(readings, reading)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}

func getMonthlyReadingsReport(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		monthStr = time.Now().Format("2006-01")
	}

	parsedMonth, err := time.Parse("2006-01", monthStr)
	if err != nil {
		http.Error(w, "Invalid month format. Use YYYY-MM", http.StatusBadRequest)
		return
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	monthlyReadings := make([]Reading, 0)
	totalConsumption := decimal.Zero

	for _, reading := range readingsDB {
		if reading.ReadingDate.Year() == parsedMonth.Year() &&
			reading.ReadingDate.Month() == parsedMonth.Month() &&
			reading.IsVerified {
			monthlyReadings = append(monthlyReadings, reading)
			totalConsumption = totalConsumption.Add(reading.Consumption)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"month":            monthStr,
		"total_readings":   len(monthlyReadings),
		"total_consumption": totalConsumption.String(),
		"readings":         monthlyReadings,
	})
}

// Helper functions
func publishReadingEvent(reading Reading, eventType string) {
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
		"readings",
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
		"event_type":     eventType,
		"reading_id":     reading.ID,
		"subscriber_id":  reading.SubscriberID,
		"account_number": reading.AccountNumber,
		"counter_id":     reading.CounterID,
		"current_value":  reading.CurrentValue.String(),
		"previous_value": reading.PreviousValue.String(),
		"consumption":    reading.Consumption.String(),
		"reading_date":   reading.ReadingDate.Format(time.RFC3339),
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(event)
	err = ch.PublishWithContext(
		context.Background(),
		"readings",
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

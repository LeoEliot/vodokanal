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

	"github.com/gorilla/mux"
)

// Counter represents a water meter
type Counter struct {
	ID              string    `json:"id"`
	SubscriberID    string    `json:"subscriber_id"`
	AccountNumber   string    `json:"account_number"`

	// Counter information
	SerialNumber    string    `json:"serial_number"`
	Type            string    `json:"type"`              // hot, cold, sewage
	Brand           string    `json:"brand"`
	Model           string    `json:"model,omitempty"`
	Capacity        float64   `json:"capacity"`          // Max reading value

	// Installation details
	InstallationDate time.Time `json:"installation_date"`
	InstalledBy      string    `json:"installed_by"`
	Location         string    `json:"location"`          // kitchen, bathroom, basement, etc.
	Notes            string    `json:"notes,omitempty"`

	// Verification
	VerificationDate  *time.Time `json:"verification_date,omitempty"`
	NextVerification  *time.Time `json:"next_verification,omitempty"`
	IsVerified        bool       `json:"is_verified"`

	// Status
	Status           string    `json:"status"`            // active, inactive, replaced, decommissioned
	InitialValue     float64   `json:"initial_value"`
	CurrentValue     float64   `json:"current_value"`     // Last known reading

	// Replacement info
	ReplacedBy       string    `json:"replaced_by,omitempty"`     // New counter ID
	ReplacementDate  *time.Time `json:"replacement_date,omitempty"`
	ReplacementReason string    `json:"replacement_reason,omitempty"`

	// Timestamps
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateCounterRequest represents a request to create a counter
type CreateCounterRequest struct {
	SubscriberID     string  `json:"subscriber_id"`
	SerialNumber     string  `json:"serial_number"`
	Type             string  `json:"type"`
	Brand            string  `json:"brand"`
	Model            string  `json:"model,omitempty"`
	Capacity         float64 `json:"capacity"`
	InstallationDate string  `json:"installation_date,omitempty"`
	Location         string  `json:"location"`
	Notes            string  `json:"notes,omitempty"`
	InitialValue     float64 `json:"initial_value"`
}

// UpdateCounterRequest represents a request to update a counter
type UpdateCounterRequest struct {
	Brand            string  `json:"brand,omitempty"`
	Model            string  `json:"model,omitempty"`
	Location         string  `json:"location,omitempty"`
	Notes            string  `json:"notes,omitempty"`
	CurrentValue     float64 `json:"current_value,omitempty"`
}

// ReplaceCounterRequest represents a request to replace a counter
type ReplaceCounterRequest struct {
	NewCounterID       string `json:"new_counter_id"`
	ReplacementReason  string `json:"replacement_reason"`
	ReplacedBy         string `json:"replaced_by"`
}

// VerifyCounterRequest represents a request to verify a counter
type VerifyCounterRequest struct {
	VerificationDate string `json:"verification_date"`
	NextVerification string `json:"next_verification,omitempty"`
	VerifiedBy       string `json:"verified_by"`
	Notes            string `json:"notes,omitempty"`
}

var (
	countersDB   = make(map[string]Counter)
	dbMutex      sync.RWMutex
	rabbitMQURL  = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize with sample counters
	installationDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	nextVerification := time.Now().AddDate(4, 0, 0)

	countersDB["counter_hot_1"] = Counter{
		ID:               "counter_hot_1",
		SubscriberID:     "sub_test",
		AccountNumber:    "000123456",
		SerialNumber:     "SN-HOT-123456",
		Type:             "hot",
		Brand:            "Эльстер",
		Model:            "Metris T4",
		Capacity:         99999.0,
		InstallationDate: installationDate,
		InstalledBy:      "emp_1",
		Location:         "Кухня",
		NextVerification: &nextVerification,
		IsVerified:       true,
		Status:           "active",
		InitialValue:     0,
		CurrentValue:     1050,
		CreatedAt:        installationDate,
		UpdatedAt:        time.Now(),
	}

	countersDB["counter_cold_1"] = Counter{
		ID:               "counter_cold_1",
		SubscriberID:     "sub_test",
		AccountNumber:    "000123456",
		SerialNumber:     "SN-COLD-789012",
		Type:             "cold",
		Brand:            "Эльстер",
		Model:            "Metris T4",
		Capacity:         99999.0,
		InstallationDate: installationDate,
		InstalledBy:      "emp_1",
		Location:         "Подвал",
		NextVerification: &nextVerification,
		IsVerified:       true,
		Status:           "active",
		InitialValue:     0,
		CurrentValue:     2500,
		CreatedAt:        installationDate,
		UpdatedAt:        time.Now(),
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Counter endpoints
	api.HandleFunc("/counters", getCounters).Methods("GET")
	api.HandleFunc("/counters/{id}", getCounter).Methods("GET")
	api.HandleFunc("/counters", createCounter).Methods("POST")
	api.HandleFunc("/counters/{id}", updateCounter).Methods("PUT")
	api.HandleFunc("/counters/{id}", deleteCounter).Methods("DELETE")
	api.HandleFunc("/counters/{id}/replace", replaceCounter).Methods("POST")
	api.HandleFunc("/counters/{id}/verify", verifyCounter).Methods("POST")
	api.HandleFunc("/counters/{id}/decommission", decommissionCounter).Methods("POST")
	api.HandleFunc("/counters/{id}/activate", activateCounter).Methods("POST")
	api.HandleFunc("/subscribers/{subscriberId}/counters", getSubscriberCounters).Methods("GET")

	// Report endpoints
	api.HandleFunc("/counters/report/due-verification", getDueVerificationCounters).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	srv := &http.Server{
		Addr:         ":8087",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Counters Service starting on port 8087")
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
		"service":   "counters",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Counter handlers
func getCounters(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	// Parse query parameters
	subscriberID := r.URL.Query().Get("subscriber_id")
	counterType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	counters := make([]Counter, 0, len(countersDB))
	for _, counter := range countersDB {
		// Apply filters
		if subscriberID != "" && counter.SubscriberID != subscriberID {
			continue
		}
		if counterType != "" && counter.Type != counterType {
			continue
		}
		if status != "" && counter.Status != status {
			continue
		}
		counters = append(counters, counter)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counters)
}

func getCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	counter, exists := countersDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counter)
}

func createCounter(w http.ResponseWriter, r *http.Request) {
	var req CreateCounterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SubscriberID == "" || req.SerialNumber == "" || req.Type == "" || req.Brand == "" {
		http.Error(w, "Subscriber ID, serial number, type, and brand are required", http.StatusBadRequest)
		return
	}

	// Validate counter type
	if req.Type != "hot" && req.Type != "cold" && req.Type != "sewage" {
		http.Error(w, "Type must be one of: hot, cold, sewage", http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check for duplicate serial number
	for _, c := range countersDB {
		if c.SerialNumber == req.SerialNumber && c.Status != "decommissioned" {
			http.Error(w, "Serial number already in use", http.StatusConflict)
			return
		}
	}

	// Generate unique ID
	id := fmt.Sprintf("counter_%s_%d", req.Type, time.Now().UnixNano())

	// Parse installation date
	var installationDate time.Time
	if req.InstallationDate != "" {
		parsedDate, err := time.Parse(time.RFC3339, req.InstallationDate)
		if err != nil {
			http.Error(w, "Invalid installation date format", http.StatusBadRequest)
			return
		}
		installationDate = parsedDate
	} else {
		installationDate = time.Now()
	}

	// Get account number (in real implementation, fetch from subscribers service)
	accountNumber := "000000000"
	for _, c := range countersDB {
		if c.SubscriberID == req.SubscriberID {
			accountNumber = c.AccountNumber
			break
		}
	}

	// Set default capacity
	capacity := req.Capacity
	if capacity == 0 {
		capacity = 99999.0
	}

	// Set next verification date (4 years from installation)
	nextVerification := installationDate.AddDate(4, 0, 0)

	counter := Counter{
		ID:               id,
		SubscriberID:     req.SubscriberID,
		AccountNumber:    accountNumber,
		SerialNumber:     req.SerialNumber,
		Type:             req.Type,
		Brand:            req.Brand,
		Model:            req.Model,
		Capacity:         capacity,
		InstallationDate: installationDate,
		InstalledBy:      "system",
		Location:         req.Location,
		Notes:            req.Notes,
		NextVerification: &nextVerification,
		IsVerified:       true,
		Status:           "active",
		InitialValue:     req.InitialValue,
		CurrentValue:     req.InitialValue,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	countersDB[id] = counter

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(counter)
}

func updateCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateCounterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	counter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if req.Brand != "" {
		counter.Brand = req.Brand
	}
	if req.Model != "" {
		counter.Model = req.Model
	}
	if req.Location != "" {
		counter.Location = req.Location
	}
	if req.Notes != "" {
		counter.Notes = req.Notes
	}
	if req.CurrentValue >= 0 {
		counter.CurrentValue = req.CurrentValue
	}

	counter.UpdatedAt = time.Now()
	countersDB[id] = counter

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counter)
}

func deleteCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	counter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	// Only allow deletion of decommissioned counters
	if counter.Status != "decommissioned" {
		http.Error(w, "Cannot delete active counter. Decommission it first.", http.StatusBadRequest)
		return
	}

	delete(countersDB, id)
	w.WriteHeader(http.StatusNoContent)
}

func replaceCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req ReplaceCounterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	oldCounter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	// Check if new counter exists
	newCounter, newExists := countersDB[req.NewCounterID]
	if !newExists {
		http.Error(w, "New counter not found", http.StatusNotFound)
		return
	}

	// Mark old counter as replaced
	now := time.Now()
	oldCounter.Status = "replaced"
	oldCounter.ReplacedBy = req.NewCounterID
	oldCounter.ReplacementDate = &now
	oldCounter.ReplacementReason = req.ReplacementReason
	countersDB[id] = oldCounter

	// Update new counter installation info
	newCounter.InstallationDate = now
	newCounter.InstalledBy = req.ReplacedBy
	newCounter.InitialValue = oldCounter.CurrentValue
	newCounter.CurrentValue = oldCounter.CurrentValue
	countersDB[req.NewCounterID] = newCounter

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"old_counter": oldCounter,
		"new_counter": newCounter,
	})
}

func verifyCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req VerifyCounterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	counter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	// Parse verification date
	verificationDate, err := time.Parse(time.RFC3339, req.VerificationDate)
	if err != nil {
		http.Error(w, "Invalid verification date format", http.StatusBadRequest)
		return
	}

	var nextVerification *time.Time
	if req.NextVerification != "" {
		nextDate, err := time.Parse(time.RFC3339, req.NextVerification)
		if err != nil {
			http.Error(w, "Invalid next verification date format", http.StatusBadRequest)
			return
		}
		nextVerification = &nextDate
	} else {
		// Default: 4 years from verification
		nextDate := verificationDate.AddDate(4, 0, 0)
		nextVerification = &nextDate
	}

	counter.VerificationDate = &verificationDate
	counter.NextVerification = nextVerification
	counter.IsVerified = true
	counter.UpdatedAt = time.Now()

	if req.Notes != "" {
		if counter.Notes != "" {
			counter.Notes = counter.Notes + "\n" + req.Notes
		} else {
			counter.Notes = req.Notes
		}
	}

	countersDB[id] = counter

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counter)
}

func decommissionCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	counter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	counter.Status = "decommissioned"
	counter.UpdatedAt = time.Now()
	if req.Reason != "" {
		if counter.Notes != "" {
			counter.Notes = counter.Notes + "\nDecommissioned: " + req.Reason
		} else {
			counter.Notes = "Decommissioned: " + req.Reason
		}
	}
	countersDB[id] = counter

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counter)
}

func activateCounter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	counter, exists := countersDB[id]
	if !exists {
		http.Error(w, "Counter not found", http.StatusNotFound)
		return
	}

	if counter.Status == "active" {
		http.Error(w, "Counter is already active", http.StatusBadRequest)
		return
	}

	counter.Status = "active"
	counter.UpdatedAt = time.Now()
	countersDB[id] = counter

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counter)
}

func getSubscriberCounters(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriberID := vars["subscriberId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	counters := make([]Counter, 0)
	for _, counter := range countersDB {
		if counter.SubscriberID == subscriberID {
			counters = append(counters, counter)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counters)
}

func getDueVerificationCounters(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	now := time.Now()
	dueCounters := make([]Counter, 0)

	for _, counter := range countersDB {
		if counter.Status == "active" && counter.NextVerification != nil {
			if counter.NextVerification.Before(now) ||
				counter.NextVerification.Before(now.AddDate(0, 1, 0)) { // Due within 1 month
				dueCounters = append(dueCounters, counter)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_due": len(dueCounters),
		"counters":  dueCounters,
	})
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

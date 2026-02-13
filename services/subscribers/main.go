package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Subscriber represents a water utility subscriber
type Subscriber struct {
	ID              string    `json:"id"`
	AccountNumber   string    `json:"account_number"`   // Unique account number

	// Personal information
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Patronymic      string    `json:"patronymic,omitempty"`

	// Contact information
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	AlternatePhone  string    `json:"alternate_phone,omitempty"`

	// Address
	Address         string    `json:"address"`
	City            string    `json:"city"`
	PostalCode      string    `json:"postal_code"`
	Apartment       string    `json:"apartment,omitempty"`

	// Service details
	ServiceType     string    `json:"service_type"`      // residential, commercial
	ContractDate    time.Time `json:"contract_date"`
	IsActive        bool      `json:"is_active"`

	// Account status
	Status          string    `json:"status"`            // active, suspended, closed
	SuspensionReason string   `json:"suspension_reason,omitempty"`

	// Timestamps
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

// CreateSubscriberRequest represents a request to create a subscriber
type CreateSubscriberRequest struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Patronymic     string `json:"patronymic,omitempty"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	AlternatePhone string `json:"alternate_phone,omitempty"`
	Address        string `json:"address"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Apartment      string `json:"apartment,omitempty"`
	ServiceType    string `json:"service_type"`
}

var (
	subscribersDB = make(map[string]Subscriber)
	dbMutex       sync.RWMutex
	rabbitMQURL   = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize with sample subscribers
	subscribersDB["sub_test"] = Subscriber{
		ID:            "sub_test",
		AccountNumber: "000123456",
		FirstName:     "Иван",
		LastName:      "Иванов",
		Patronymic:    "Иванович",
		Email:         "ivanov@example.com",
		Phone:         "+7-999-123-45-67",
		Address:       "ул. Ленина, д. 1",
		City:          "Москва",
		PostalCode:    "123456",
		Apartment:     "10",
		ServiceType:   "residential",
		ContractDate:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		IsActive:      true,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Subscriber endpoints
	api.HandleFunc("/subscribers", getSubscribers).Methods("GET")
	api.HandleFunc("/subscribers/{id}", getSubscriber).Methods("GET")
	api.HandleFunc("/subscribers", createSubscriber).Methods("POST")
	api.HandleFunc("/subscribers/{id}", updateSubscriber).Methods("PUT")
	api.HandleFunc("/subscribers/{id}", deleteSubscriber).Methods("DELETE")
	api.HandleFunc("/subscribers/{id}/suspend", suspendSubscriber).Methods("POST")
	api.HandleFunc("/subscribers/{id}/activate", activateSubscriber).Methods("POST")
	api.HandleFunc("/subscribers/account/{accountNumber}", getSubscriberByAccount).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	srv := &http.Server{
		Addr:         ":8082",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Subscribers Service starting on port 8082")
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
		"service":   "subscribers",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Subscriber handlers
func getSubscribers(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	// Parse query parameters for filtering
	status := r.URL.Query().Get("status")
	serviceType := r.URL.Query().Get("service_type")
	search := r.URL.Query().Get("search")

	subscribers := make([]Subscriber, 0, len(subscribersDB))
	for _, s := range subscribersDB {
		// Apply filters
		if status != "" && s.Status != status {
			continue
		}
		if serviceType != "" && s.ServiceType != serviceType {
			continue
		}
		if search != "" {
			searchLower := strings.ToLower(search)
			fullName := strings.ToLower(s.LastName + " " + s.FirstName + " " + s.Patronymic)
			accountNum := strings.ToLower(s.AccountNumber)
			email := strings.ToLower(s.Email)
			if !strings.Contains(fullName, searchLower) &&
				!strings.Contains(accountNum, searchLower) &&
				!strings.Contains(email, searchLower) {
				continue
			}
		}
		subscribers = append(subscribers, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscribers)
}

func getSubscriber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	subscriber, exists := subscribersDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Subscriber not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriber)
}

func createSubscriber(w http.ResponseWriter, r *http.Request) {
	var req CreateSubscriberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.FirstName == "" || req.LastName == "" || req.Email == "" || req.Phone == "" {
		http.Error(w, "First name, last name, email, and phone are required", http.StatusBadRequest)
		return
	}

	if req.Address == "" || req.City == "" {
		http.Error(w, "Address and city are required", http.StatusBadRequest)
		return
	}

	// Validate email format
	if !strings.Contains(req.Email, "@") {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check for duplicate email
	for _, s := range subscribersDB {
		if strings.ToLower(s.Email) == strings.ToLower(req.Email) {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
	}

	// Generate unique ID and account number
	id := fmt.Sprintf("sub_%d", time.Now().UnixNano())
	accountNumber := generateAccountNumber()

	// Set default service type
	serviceType := req.ServiceType
	if serviceType == "" {
		serviceType = "residential"
	}

	subscriber := Subscriber{
		ID:             id,
		AccountNumber:  accountNumber,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Patronymic:     req.Patronymic,
		Email:          req.Email,
		Phone:          req.Phone,
		AlternatePhone: req.AlternatePhone,
		Address:        req.Address,
		City:           req.City,
		PostalCode:     req.PostalCode,
		Apartment:      req.Apartment,
		ServiceType:    serviceType,
		ContractDate:   time.Now(),
		IsActive:       true,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	subscribersDB[id] = subscriber

	// Send notification about new subscriber
	go sendSubscriberNotification(subscriber, "created")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subscriber)
}

func updateSubscriber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updatedSubscriber Subscriber
	if err := json.NewDecoder(r.Body).Decode(&updatedSubscriber); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	subscriber, exists := subscribersDB[id]
	if !exists {
		http.Error(w, "Subscriber not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if updatedSubscriber.FirstName != "" {
		subscriber.FirstName = updatedSubscriber.FirstName
	}
	if updatedSubscriber.LastName != "" {
		subscriber.LastName = updatedSubscriber.LastName
	}
	if updatedSubscriber.Patronymic != "" {
		subscriber.Patronymic = updatedSubscriber.Patronymic
	}
	if updatedSubscriber.Email != "" {
		// Check for duplicate email
		for _, s := range subscribersDB {
			if s.ID != id && strings.ToLower(s.Email) == strings.ToLower(updatedSubscriber.Email) {
				http.Error(w, "Email already registered", http.StatusConflict)
				return
			}
		}
		subscriber.Email = updatedSubscriber.Email
	}
	if updatedSubscriber.Phone != "" {
		subscriber.Phone = updatedSubscriber.Phone
	}
	if updatedSubscriber.AlternatePhone != "" {
		subscriber.AlternatePhone = updatedSubscriber.AlternatePhone
	}
	if updatedSubscriber.Address != "" {
		subscriber.Address = updatedSubscriber.Address
	}
	if updatedSubscriber.City != "" {
		subscriber.City = updatedSubscriber.City
	}
	if updatedSubscriber.PostalCode != "" {
		subscriber.PostalCode = updatedSubscriber.PostalCode
	}
	if updatedSubscriber.Apartment != "" {
		subscriber.Apartment = updatedSubscriber.Apartment
	}

	subscriber.UpdatedAt = time.Now()
	subscribersDB[id] = subscriber

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriber)
}

func deleteSubscriber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	subscriber, exists := subscribersDB[id]
	if !exists {
		http.Error(w, "Subscriber not found", http.StatusNotFound)
		return
	}

	// Soft delete - mark as closed
	now := time.Now()
	subscriber.IsActive = false
	subscriber.Status = "closed"
	subscriber.ClosedAt = &now
	subscriber.UpdatedAt = now
	subscribersDB[id] = subscriber

	// Send notification
	go sendSubscriberNotification(subscriber, "deleted")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriber)
}

func suspendSubscriber(w http.ResponseWriter, r *http.Request) {
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

	subscriber, exists := subscribersDB[id]
	if !exists {
		http.Error(w, "Subscriber not found", http.StatusNotFound)
		return
	}

	subscriber.Status = "suspended"
	subscriber.SuspensionReason = req.Reason
	subscriber.IsActive = false
	subscriber.UpdatedAt = time.Now()
	subscribersDB[id] = subscriber

	// Send notification
	go sendSubscriberNotification(subscriber, "suspended")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriber)
}

func activateSubscriber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	subscriber, exists := subscribersDB[id]
	if !exists {
		http.Error(w, "Subscriber not found", http.StatusNotFound)
		return
	}

	subscriber.Status = "active"
	subscriber.SuspensionReason = ""
	subscriber.IsActive = true
	subscriber.UpdatedAt = time.Now()
	subscribersDB[id] = subscriber

	// Send notification
	go sendSubscriberNotification(subscriber, "activated")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriber)
}

func getSubscriberByAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountNumber := vars["accountNumber"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	for _, s := range subscribersDB {
		if s.AccountNumber == accountNumber {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(s)
			return
		}
	}

	http.Error(w, "Subscriber not found", http.StatusNotFound)
}

// Helper functions
func generateAccountNumber() string {
	// Generate a unique 9-digit account number
	// In production, this would use a more sophisticated algorithm
	timestamp := time.Now().Format("20060102150405")
	return timestamp[:9]
}

func sendSubscriberNotification(subscriber Subscriber, event string) {
	// In real implementation, publish to RabbitMQ for notifications service
	log.Printf("Subscriber notification: %s - %s %s (%s)",
		event, subscriber.FirstName, subscriber.LastName, subscriber.AccountNumber)
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

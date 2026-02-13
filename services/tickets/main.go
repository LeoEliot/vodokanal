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
)

// Ticket represents a support ticket
type Ticket struct {
	ID             string       `json:"id"`
	SubscriberID   string       `json:"subscriber_id"`
	AccountNumber  string       `json:"account_number"`
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	Category       string       `json:"category"`    // leak, repair, billing, quality, other
	Priority       string       `json:"priority"`    // low, medium, high, urgent
	Status         string       `json:"status"`      // open, in_progress, resolved, closed, cancelled

	// Address
	Address        string       `json:"address"`
	Apartment      string       `json:"apartment,omitempty"`

	// Assignment
	AssignedTo     string       `json:"assigned_to,omitempty"`     // employee ID
	AssignedAt     *time.Time   `json:"assigned_at,omitempty"`

	// Resolution
	Resolution     string       `json:"resolution,omitempty"`
	ResolvedAt     *time.Time   `json:"resolved_at,omitempty"`
	ResolvedBy     string       `json:"resolved_by,omitempty"`

	// Scheduling
	ScheduledDate  *time.Time   `json:"scheduled_date,omitempty"`
	ScheduledTime  string       `json:"scheduled_time,omitempty"`   // morning, afternoon, evening

	// Contact
	ContactName    string       `json:"contact_name"`
	ContactPhone   string       `json:"contact_phone"`
	ContactEmail   string       `json:"contact_email,omitempty"`

	// Attachments
	Attachments    []Attachment `json:"attachments,omitempty"`

	// Feedback
	Rating         int          `json:"rating,omitempty"`           // 1-5
	Feedback       string       `json:"feedback,omitempty"`

	// Timestamps
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ClosedAt       *time.Time   `json:"closed_at,omitempty"`

	// Metadata
	Source         string       `json:"source"`                     // web, mobile, phone, email
	IPAddress      string       `json:"ip_address,omitempty"`
}

// Attachment represents a file attached to a ticket
type Attachment struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	URL         string    `json:"url"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Comment represents a comment on a ticket
type Comment struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	Content     string    `json:"content"`
	AuthorID    string    `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	AuthorRole  string    `json:"author_role"`  // subscriber, employee, admin
	IsInternal  bool      `json:"is_internal"`  // true for internal comments only visible to staff
	CreatedAt   time.Time `json:"created_at"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// TicketHistory represents a history entry for a ticket
type TicketHistory struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticket_id"`
	Field     string    `json:"field"`        // status, priority, assigned_to, etc.
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
}

// Employee represents a support employee
type Employee struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Role        string    `json:"role"`        // manager, operator, technician
	IsActive    bool      `json:"is_active"`
	Skills      []string  `json:"skills"`      // leak, repair, billing, etc.
	CreatedAt   time.Time `json:"created_at"`
}

var (
	ticketsDB     = make(map[string]Ticket)
	commentsDB    = make(map[string]Comment)
	historyDB     = make(map[string][]TicketHistory)
	employeesDB   = make(map[string]Employee)
	dbMutex       sync.RWMutex
	rabbitMQURL   = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize default employees
	employeesDB["emp_1"] = Employee{
		ID:        "emp_1",
		Name:      "Иванов Иван Иванович",
		Email:     "ivanov@vodokanal.ru",
		Phone:     "+7-999-123-45-67",
		Role:      "manager",
		IsActive:  true,
		Skills:    []string{"leak", "repair", "billing", "quality"},
		CreatedAt: time.Now(),
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Ticket endpoints
	api.HandleFunc("/tickets", getTickets).Methods("GET")
	api.HandleFunc("/tickets/{id}", getTicket).Methods("GET")
	api.HandleFunc("/tickets", createTicket).Methods("POST")
	api.HandleFunc("/tickets/{id}", updateTicket).Methods("PUT")
	api.HandleFunc("/tickets/{id}", deleteTicket).Methods("DELETE")
	api.HandleFunc("/tickets/{id}/close", closeTicket).Methods("POST")
	api.HandleFunc("/tickets/{id}/reopen", reopenTicket).Methods("POST")
	api.HandleFunc("/tickets/{id}/assign", assignTicket).Methods("POST")
	api.HandleFunc("/subscribers/{subscriberId}/tickets", getSubscriberTickets).Methods("GET")

	// Comment endpoints
	api.HandleFunc("/tickets/{id}/comments", getTicketComments).Methods("GET")
	api.HandleFunc("/tickets/{id}/comments", addComment).Methods("POST")
	api.HandleFunc("/comments/{id}", updateComment).Methods("PUT")
	api.HandleFunc("/comments/{id}", deleteComment).Methods("DELETE")

	// History endpoints
	api.HandleFunc("/tickets/{id}/history", getTicketHistory).Methods("GET")

	// Employee endpoints
	api.HandleFunc("/employees", getEmployees).Methods("GET")
	api.HandleFunc("/employees/{id}", getEmployee).Methods("GET")
	api.HandleFunc("/employees", createEmployee).Methods("POST")
	api.HandleFunc("/employees/{id}", updateEmployee).Methods("PUT")
	api.HandleFunc("/employees/{id}", deleteEmployee).Methods("DELETE")

	// Statistics endpoints
	api.HandleFunc("/stats/tickets", getTicketsStats).Methods("GET")
	api.HandleFunc("/stats/employees", getEmployeesStats).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	// Start RabbitMQ consumer for notifications
	go startRabbitMQConsumer()

	srv := &http.Server{
		Addr:         ":8086",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Tickets Service starting on port 8086")
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
		"service":   "tickets",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Ticket handlers
func getTickets(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	// Parse query parameters
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	category := r.URL.Query().Get("category")
	assignedTo := r.URL.Query().Get("assigned_to")

	tickets := make([]Ticket, 0, len(ticketsDB))
	for _, t := range ticketsDB {
		if status != "" && t.Status != status {
			continue
		}
		if priority != "" && t.Priority != priority {
			continue
		}
		if category != "" && t.Category != category {
			continue
		}
		if assignedTo != "" && t.AssignedTo != assignedTo {
			continue
		}
		tickets = append(tickets, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

func getTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	ticket, exists := ticketsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func createTicket(w http.ResponseWriter, r *http.Request) {
	var ticket Ticket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ticket.ID = fmt.Sprintf("ticket_%d", time.Now().UnixNano())
	ticket.Status = "open"

	if ticket.Priority == "" {
		ticket.Priority = "medium"
	}

	if ticket.Source == "" {
		ticket.Source = "web"
	}

	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	dbMutex.Lock()
	ticketsDB[ticket.ID] = ticket
	historyDB[ticket.ID] = []TicketHistory{
		{
			ID:        fmt.Sprintf("hist_%d", time.Now().UnixNano()),
			TicketID:  ticket.ID,
			Field:     "status",
			OldValue:  "",
			NewValue:  "open",
			ChangedBy: ticket.SubscriberID,
			ChangedAt: time.Now(),
		},
	}
	dbMutex.Unlock()

	// Send notification
	go sendTicketNotification(ticket, "created")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ticket)
}

func updateTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updatedTicket Ticket
	if err := json.NewDecoder(r.Body).Decode(&updatedTicket); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	ticket, exists := ticketsDB[id]
	if !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	// Track changes
	_  = ticket

	// Update fields
	if updatedTicket.Title != "" && updatedTicket.Title != ticket.Title {
		addHistoryEntry(id, "title", ticket.Title, updatedTicket.Title, updatedTicket.SubscriberID)
		ticket.Title = updatedTicket.Title
	}
	if updatedTicket.Description != "" && updatedTicket.Description != ticket.Description {
		addHistoryEntry(id, "description", ticket.Description, updatedTicket.Description, updatedTicket.SubscriberID)
		ticket.Description = updatedTicket.Description
	}
	if updatedTicket.Priority != "" && updatedTicket.Priority != ticket.Priority {
		addHistoryEntry(id, "priority", ticket.Priority, updatedTicket.Priority, updatedTicket.SubscriberID)
		ticket.Priority = updatedTicket.Priority
	}
	if updatedTicket.Category != "" && updatedTicket.Category != ticket.Category {
		addHistoryEntry(id, "category", ticket.Category, updatedTicket.Category, updatedTicket.SubscriberID)
		ticket.Category = updatedTicket.Category
	}
	if updatedTicket.Address != "" && updatedTicket.Address != ticket.Address {
		ticket.Address = updatedTicket.Address
	}
	if updatedTicket.Apartment != "" && updatedTicket.Apartment != ticket.Apartment {
		ticket.Apartment = updatedTicket.Apartment
	}
	if updatedTicket.ContactPhone != "" && updatedTicket.ContactPhone != ticket.ContactPhone {
		ticket.ContactPhone = updatedTicket.ContactPhone
	}
	if updatedTicket.ContactEmail != "" && updatedTicket.ContactEmail != ticket.ContactEmail {
		ticket.ContactEmail = updatedTicket.ContactEmail
	}

	ticket.UpdatedAt = time.Now()
	ticketsDB[id] = ticket

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func deleteTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := ticketsDB[id]; !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	delete(ticketsDB, id)
	delete(historyDB, id)

	w.WriteHeader(http.StatusNoContent)
}

func closeTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data struct {
		Resolution string `json:"resolution"`
		Rating     int    `json:"rating"`
		Feedback   string `json:"feedback"`
	}
	json.NewDecoder(r.Body).Decode(&data)

	dbMutex.Lock()
	defer dbMutex.Unlock()

	ticket, exists := ticketsDB[id]
	if !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if ticket.Status == "closed" {
		http.Error(w, "Ticket already closed", http.StatusBadRequest)
		return
	}

	oldStatus := ticket.Status
	ticket.Status = "closed"
	ticket.Resolution = data.Resolution
	ticket.Rating = data.Rating
	ticket.Feedback = data.Feedback
	now := time.Now()
	ticket.ClosedAt = &now
	ticket.ResolvedAt = &now
	ticket.UpdatedAt = now

	ticketsDB[id] = ticket

	addHistoryEntry(id, "status", oldStatus, "closed", ticket.SubscriberID)

	// Send notification
	go sendTicketNotification(ticket, "closed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func reopenTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	ticket, exists := ticketsDB[id]
	if !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if ticket.Status != "closed" && ticket.Status != "resolved" {
		http.Error(w, "Can only reopen closed or resolved tickets", http.StatusBadRequest)
		return
	}

	oldStatus := ticket.Status
	ticket.Status = "open"
	ticket.UpdatedAt = time.Now()

	ticketsDB[id] = ticket

	addHistoryEntry(id, "status", oldStatus, "open", ticket.SubscriberID)

	// Send notification
	go sendTicketNotification(ticket, "reopened")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func assignTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	ticket, exists := ticketsDB[id]
	if !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if _, empExists := employeesDB[data.EmployeeID]; !empExists {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	oldAssignedTo := ticket.AssignedTo
	ticket.AssignedTo = data.EmployeeID
	ticket.Status = "in_progress"
	now := time.Now()
	ticket.AssignedAt = &now
	ticket.UpdatedAt = now

	ticketsDB[id] = ticket

	addHistoryEntry(id, "assigned_to", oldAssignedTo, data.EmployeeID, data.EmployeeID)
	addHistoryEntry(id, "status", "open", "in_progress", data.EmployeeID)

	// Send notification
	go sendTicketNotification(ticket, "assigned")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func getSubscriberTickets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriberID := vars["subscriberId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	subscriberTickets := make([]Ticket, 0)
	for _, t := range ticketsDB {
		if t.SubscriberID == subscriberID {
			subscriberTickets = append(subscriberTickets, t)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriberTickets)
}

// Comment handlers
func getTicketComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if _, exists := ticketsDB[id]; !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	comments := make([]Comment, 0)
	for _, c := range commentsDB {
		if c.TicketID == id && !c.IsInternal {
			comments = append(comments, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func addComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var comment Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := ticketsDB[id]; !exists {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	comment.ID = fmt.Sprintf("comment_%d", time.Now().UnixNano())
	comment.TicketID = id
	comment.CreatedAt = time.Now()

	commentsDB[comment.ID] = comment

	// Send notification
	go sendCommentNotification(comment, ticketsDB[id])

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func updateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var comment Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := commentsDB[id]; !exists {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	comment.ID = id
	commentsDB[id] = comment

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func deleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := commentsDB[id]; !exists {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	delete(commentsDB, id)
	w.WriteHeader(http.StatusNoContent)
}

// History handlers
func getTicketHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	history, exists := historyDB[id]
	dbMutex.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]TicketHistory{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// Employee handlers
func getEmployees(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	employees := make([]Employee, 0, len(employeesDB))
	for _, e := range employeesDB {
		employees = append(employees, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(employees)
}

func getEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	employee, exists := employeesDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(employee)
}

func createEmployee(w http.ResponseWriter, r *http.Request) {
	var employee Employee
	if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	employee.ID = fmt.Sprintf("emp_%d", time.Now().UnixNano())
	employee.CreatedAt = time.Now()

	dbMutex.Lock()
	employeesDB[employee.ID] = employee
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(employee)
}

func updateEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var employee Employee
	if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := employeesDB[id]; !exists {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	employee.ID = id
	employeesDB[id] = employee

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(employee)
}

func deleteEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := employeesDB[id]; !exists {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	delete(employeesDB, id)
	w.WriteHeader(http.StatusNoContent)
}

// Statistics handlers
func getTicketsStats(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	stats := map[string]int{
		"total":      len(ticketsDB),
		"open":       0,
		"in_progress": 0,
		"resolved":   0,
		"closed":     0,
		"cancelled":  0,
	}

	priorityStats := map[string]int{
		"low":    0,
		"medium": 0,
		"high":   0,
		"urgent": 0,
	}

	for _, t := range ticketsDB {
		stats[t.Status]++
		priorityStats[t.Priority]++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"by_status":   stats,
		"by_priority": priorityStats,
	})
}

func getEmployeesStats(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	employeeStats := make([]map[string]interface{}, 0)

	for _, emp := range employeesDB {
		if !emp.IsActive {
			continue
		}

		assignedCount := 0
		openCount := 0
		closedCount := 0

		for _, t := range ticketsDB {
			if t.AssignedTo == emp.ID {
				assignedCount++
				if t.Status == "open" || t.Status == "in_progress" {
					openCount++
				}
				if t.Status == "closed" {
					closedCount++
				}
			}
		}

		employeeStats = append(employeeStats, map[string]interface{}{
			"employee_id":     emp.ID,
			"name":           emp.Name,
			"role":           emp.Role,
			"assigned_total":  assignedCount,
			"assigned_open":   openCount,
			"assigned_closed": closedCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(employeeStats)
}

// Helper functions
func addHistoryEntry(ticketID, field, oldValue, newValue, changedBy string) {
	entry := TicketHistory{
		ID:        fmt.Sprintf("hist_%d", time.Now().UnixNano()),
		TicketID:  ticketID,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}
	historyDB[ticketID] = append(historyDB[ticketID], entry)
}

func sendTicketNotification(ticket Ticket, event string) {
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

	notificationData := map[string]interface{}{
		"channel":       "ticket_update",
		"user_id":       ticket.SubscriberID,
		"subscriber_id": ticket.SubscriberID,
		"recipient":     ticket.ContactEmail,
		"ticket_id":     ticket.ID,
		"status":        ticket.Status,
		"title":         ticket.Title,
		"name":          ticket.ContactName,
		"comment":       "",
		"event":         event,
	}

	body, _ := json.Marshal(notificationData)

	err = ch.PublishWithContext(
		context.Background(),
		"notifications",
		"notification",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish notification: %v", err)
	}
}

func sendCommentNotification(comment Comment, ticket Ticket) {
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

	notificationData := map[string]interface{}{
		"channel":       "ticket_update",
		"user_id":       ticket.SubscriberID,
		"subscriber_id": ticket.SubscriberID,
		"recipient":     ticket.ContactEmail,
		"ticket_id":     ticket.ID,
		"status":        ticket.Status,
		"title":         ticket.Title,
		"name":          ticket.ContactName,
		"comment":       fmt.Sprintf("Новый комментарий от %s", comment.AuthorName),
	}

	body, _ := json.Marshal(notificationData)

	err = ch.PublishWithContext(
		context.Background(),
		"notifications",
		"notification",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish notification: %v", err)
	}
}

func startRabbitMQConsumer() {
	// Consumer for handling responses from other services
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

		q, err := ch.QueueDeclare(
			"tickets_queue",
			true,
			false,
			false,
			false,
			nil,
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
			false,
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

		log.Println("Tickets RabbitMQ consumer started")

		for d := range msgs {
			// Process messages from other services
			d.Ack(false)
		}

		ch.Close()
		conn.Close()
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

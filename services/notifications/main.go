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
	"gopkg.in/gomail.v2"
)

// Notification represents a notification entity
type Notification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	SubscriberID string   `json:"subscriber_id,omitempty"`
	Type        string    `json:"type"` // email, sms, push
	Channel     string    `json:"channel"` // payment_reminder, reading_reminder, ticket_update, bill_generated
	Subject     string    `json:"subject,omitempty"`
	Body        string    `json:"body"`
	Recipient   string    `json:"recipient"` // email or phone
	Status      string    `json:"status"` // pending, sent, failed
	CreatedAt   time.Time `json:"created_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID        string `json:"id"`
	Channel   string `json:"channel"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Variables []string `json:"variables"` // e.g. ["{{.Name}}", "{{.Amount}}"]
}

var (
	notificationsDB = make(map[string]Notification)
	templatesDB     = make(map[string]NotificationTemplate)
	dbMutex         sync.RWMutex
	smtpHost        = os.Getenv("SMTP_HOST")
	smtpPort        = 587
	smtpUser        = os.Getenv("SMTP_USER")
	smtpPassword    = os.Getenv("SMTP_PASSWORD")
	smtpFrom        = os.Getenv("SMTP_FROM")
	rabbitMQURL     = os.Getenv("RABBITMQ_URL")
)

func init() {
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}
	if smtpHost == "" {
		smtpHost = "localhost"
	}
	if smtpFrom == "" {
		smtpFrom = "noreply@vodokanal.ru"
	}

	// Initialize default templates
	templatesDB["payment_reminder"] = NotificationTemplate{
		ID:      "payment_reminder",
		Channel: "email",
		Subject: "Напоминание о оплате",
		Body:    "Уважаемый{{if .Name}} {{.Name}}{{end}}! Напоминаем, что у вас есть неоплаченный счет на {{.Amount}} руб. Срок оплаты: {{.DueDate}}.",
		Variables: []string{"Name", "Amount", "DueDate"},
	}

	templatesDB["reading_reminder"] = NotificationTemplate{
		ID:      "reading_reminder",
		Channel: "email",
		Subject: "Напоминание о передаче показаний",
		Body:    "Уважаемый{{if .Name}} {{.Name}}{{end}}! Пожалуйста, передайте показания счетчиков воды до {{.DueDate}}. Текущий период: {{.Period}}.",
		Variables: []string{"Name", "DueDate", "Period"},
	}

	templatesDB["ticket_update"] = NotificationTemplate{
		ID:      "ticket_update",
		Channel: "email",
		Subject: "Обновление по заявке #{{.TicketID}}",
		Body:    "Уважаемый{{if .Name}} {{.Name}}{{end}}! Статус вашей заявки #{{.TicketID}} изменен на «{{.Status}}».{{if .Comment}} Комментарий: {{.Comment}}{{end}}",
		Variables: []string{"Name", "TicketID", "Status", "Comment"},
	}

	templatesDB["bill_generated"] = NotificationTemplate{
		ID:      "bill_generated",
		Channel: "email",
		Subject: "Новый счет на оплату",
		Body:    "Уважаемый{{if .Name}} {{.Name}}{{end}}! Выпущен новый счет на оплату услуг водоснабжения. Сумма: {{.Amount}} руб. Срок оплаты: {{.DueDate}}. Номер счета: {{.BillNumber}}.",
		Variables: []string{"Name", "Amount", "DueDate", "BillNumber"},
	}
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Notification management endpoints
	api.HandleFunc("/notifications", getNotifications).Methods("GET")
	api.HandleFunc("/notifications/{id}", getNotification).Methods("GET")
	api.HandleFunc("/notifications", createNotification).Methods("POST")
	api.HandleFunc("/notifications/{id}", updateNotification).Methods("PUT")
	api.HandleFunc("/notifications/{id}", deleteNotification).Methods("DELETE")
	api.HandleFunc("/notifications/user/{userId}", getUserNotifications).Methods("GET")

	// Template management endpoints
	api.HandleFunc("/templates", getTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", getTemplate).Methods("GET")
	api.HandleFunc("/templates", createTemplate).Methods("POST")
	api.HandleFunc("/templates/{id}", updateTemplate).Methods("PUT")
	api.HandleFunc("/templates/{id}", deleteTemplate).Methods("DELETE")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	// Start RabbitMQ consumer in background
	go startRabbitMQConsumer()

	// Start notification worker
	go startNotificationWorker()

	srv := &http.Server{
		Addr:         ":8088",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Notifications Service starting on port 8088")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
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

// Handler functions
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "notifications",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func getNotifications(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	notifications := make([]Notification, 0, len(notificationsDB))
	for _, n := range notificationsDB {
		notifications = append(notifications, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func getNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	notification, exists := notificationsDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notification.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	notification.Status = "pending"
	notification.CreatedAt = time.Now()

	dbMutex.Lock()
	notificationsDB[notification.ID] = notification
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

func updateNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := notificationsDB[id]; !exists {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	notification.ID = id
	notificationsDB[id] = notification

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

func deleteNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := notificationsDB[id]; !exists {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	delete(notificationsDB, id)
	w.WriteHeader(http.StatusNoContent)
}

func getUserNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	userNotifications := make([]Notification, 0)
	for _, n := range notificationsDB {
		if n.UserID == userID {
			userNotifications = append(userNotifications, n)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userNotifications)
}

// Template handlers
func getTemplates(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	templates := make([]NotificationTemplate, 0, len(templatesDB))
	for _, t := range templatesDB {
		templates = append(templates, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func getTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.RLock()
	template, exists := templatesDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func createTemplate(w http.ResponseWriter, r *http.Request) {
	var template NotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	template.ID = fmt.Sprintf("tpl_%d", time.Now().UnixNano())

	dbMutex.Lock()
	templatesDB[template.ID] = template
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func updateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var template NotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := templatesDB[id]; !exists {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	template.ID = id
	templatesDB[id] = template

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func deleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := templatesDB[id]; !exists {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	delete(templatesDB, id)
	w.WriteHeader(http.StatusNoContent)
}

// RabbitMQ Consumer
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

		// Declare exchange
		err = ch.ExchangeDeclare(
			"notifications", // name
			"direct",        // type
			true,            // durable
			false,           // auto-deleted
			false,           // internal
			false,           // no-wait
			nil,             // arguments
		)
		if err != nil {
			log.Printf("Failed to declare exchange: %v", err)
			ch.Close()
			conn.Close()
			continue
		}

		// Declare queue
		q, err := ch.QueueDeclare(
			"notifications_queue",
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

		// Bind queue to exchange
		err = ch.QueueBind(
			q.Name,          // queue name
			"notification",  // routing key
			"notifications", // exchange
			false,
			nil,
		)
		if err != nil {
			log.Printf("Failed to bind queue: %v", err)
			ch.Close()
			conn.Close()
			continue
		}

		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		if err != nil {
			log.Printf("Failed to register consumer: %v", err)
			ch.Close()
			conn.Close()
			continue
		}

		log.Println("RabbitMQ consumer started successfully")

		for d := range msgs {
			processNotificationMessage(&d)
			d.Ack(false)
		}

		ch.Close()
		conn.Close()
		log.Println("RabbitMQ connection lost, reconnecting...")
	}
}

func processNotificationMessage(msg *amqp.Delivery) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Body, &data); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	channel, _ := data["channel"].(string)
	userID, _ := data["user_id"].(string)
	subscriberID, _ := data["subscriber_id"].(string)
	recipient, _ := data["recipient"].(string) // email or phone

	// Get template
	templateID := channel
	template, exists := templatesDB[templateID]
	if !exists {
		log.Printf("Template not found: %s", templateID)
		return
	}

	// Apply template variables
	subject := template.Subject
	body := applyTemplate(template.Body, data)

	// Create notification
	notification := Notification{
		ID:           fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		UserID:       userID,
		SubscriberID: subscriberID,
		Type:         "email",
		Channel:      channel,
		Subject:      subject,
		Body:         body,
		Recipient:    recipient,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	dbMutex.Lock()
	notificationsDB[notification.ID] = notification
	dbMutex.Unlock()

	log.Printf("Notification created: %s for %s", notification.ID, recipient)
}

// Notification Worker
func startNotificationWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		processPendingNotifications()
	}
}

func processPendingNotifications() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	for id, notification := range notificationsDB {
		if notification.Status != "pending" {
			continue
		}

		var err error
		now := time.Now()

		switch notification.Type {
		case "email":
			err = sendEmail(notification)
		case "sms":
			err = sendSMS(notification)
		default:
			log.Printf("Unknown notification type: %s", notification.Type)
			continue
		}

		notification.Status = "sent"
		notification.SentAt = &now

		if err != nil {
			notification.Status = "failed"
			notification.Error = err.Error()
			log.Printf("Failed to send notification %s: %v", id, err)
		} else {
			log.Printf("Notification %s sent successfully to %s", id, notification.Recipient)
		}

		notificationsDB[id] = notification
	}
}

func sendEmail(notification Notification) error {
	m := gomail.NewMessage()
	m.SetHeader("From", smtpFrom)
	m.SetHeader("To", notification.Recipient)
	m.SetHeader("Subject", notification.Subject)
	m.SetBody("text/html", notification.Body)

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)
	return dialer.DialAndSend(m)
}

func sendSMS(notification Notification) error {
	// SMS integration would go here
	// For now, just log
	log.Printf("SMS would be sent to %s: %s", notification.Recipient, notification.Body)
	return nil
}

// Apply template variables
func applyTemplate(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		if valueStr, ok := value.(string); ok {
			result = replaceAll(result, placeholder, valueStr)
		}
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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

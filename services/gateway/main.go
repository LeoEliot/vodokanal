package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// ServiceConfig represents the configuration for a backend service
type ServiceConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
	Path string `json:"path"`
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

// GatewayConfig represents the gateway configuration
var services = map[string]ServiceConfig{
	"auth": {
		Name: "auth",
		Host: getEnv("AUTH_SERVICE_HOST", "localhost"),
		Port: getEnvInt("AUTH_SERVICE_PORT", 8081),
		Path: "/api/v1",
	},
	"subscribers": {
		Name: "subscribers",
		Host: getEnv("SUBSCRIBERS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("SUBSCRIBERS_SERVICE_PORT", 8082),
		Path: "/api/v1",
	},
	"readings": {
		Name: "readings",
		Host: getEnv("READINGS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("READINGS_SERVICE_PORT", 8083),
		Path: "/api/v1",
	},
	"billing": {
		Name: "billing",
		Host: getEnv("BILLING_SERVICE_HOST", "localhost"),
		Port: getEnvInt("BILLING_SERVICE_PORT", 8084),
		Path: "/api/v1",
	},
	"payments": {
		Name: "payments",
		Host: getEnv("PAYMENTS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("PAYMENTS_SERVICE_PORT", 8085),
		Path: "/api/v1",
	},
	"tickets": {
		Name: "tickets",
		Host: getEnv("TICKETS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("TICKETS_SERVICE_PORT", 8086),
		Path: "/api/v1",
	},
	"counters": {
		Name: "counters",
		Host: getEnv("COUNTERS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("COUNTERS_SERVICE_PORT", 8087),
		Path: "/api/v1",
	},
	"notifications": {
		Name: "notifications",
		Host: getEnv("NOTIFICATIONS_SERVICE_HOST", "localhost"),
		Port: getEnvInt("NOTIFICATIONS_SERVICE_PORT", 8088),
		Path: "/api/v1",
	},
}

// Public routes that don't require authentication
var publicRoutes = map[string]bool{
	"/api/v1/auth/login":           true,
	"/api/v1/auth/register":        true,
	"/api/v1/auth/refresh":         true,
	"/api/v1/health":               true,
	"/health":                      true,
	"/":                            true,
	"/api/v1":                      true,
}

// Routes that bypass the gateway (forwarded directly)
var directRoutes = map[string]string{
	"/api/v1/auth/":       "auth",
	"/api/v1/subscribers/": "subscribers",
	"/api/v1/readings/":   "readings",
	"/api/v1/bills/":      "billing",
	"/api/v1/tariffs/":    "billing",
	"/api/v1/reports/":    "billing",
	"/api/v1/payments/":   "payments",
	"/api/v1/refunds/":    "payments",
	"/api/v1/tickets/":    "tickets",
	"/api/v1/employees/":  "tickets",
	"/api/v1/stats/":      "tickets",
	"/api/v1/counters/":   "counters",
	"/api/v1/notifications/": "notifications",
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getServiceURL(serviceName string) string {
	service, exists := services[serviceName]
	if !exists {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", service.Host, service.Port)
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// Health check endpoint
	r.HandleFunc("/health", gatewayHealthCheck).Methods("GET")
	r.HandleFunc("/api/v1/health", gatewayHealthCheck).Methods("GET")

	// Service health check
	r.HandleFunc("/api/v1/services/health", servicesHealthCheck).Methods("GET")
	r.HandleFunc("/api/v1/services/status", servicesStatus).Methods("GET")

	// API routes - forward to appropriate services
	r.PathPrefix("/api/v1/auth/").Handler(createReverseProxy("auth"))
	r.PathPrefix("/api/v1/subscribers/").Handler(createReverseProxy("subscribers"))
	r.PathPrefix("/api/v1/readings/").Handler(createReverseProxy("readings"))
	r.PathPrefix("/api/v1/bills/").Handler(createReverseProxy("billing"))
	r.PathPrefix("/api/v1/tariffs/").Handler(createReverseProxy("billing"))
	r.PathPrefix("/api/v1/payments/").Handler(createReverseProxy("payments"))
	r.PathPrefix("/api/v1/refunds/").Handler(createReverseProxy("payments"))
	r.PathPrefix("/api/v1/payment-methods").Handler(createReverseProxy("payments"))
	r.PathPrefix("/api/v1/tickets/").Handler(createReverseProxy("tickets"))
	r.PathPrefix("/api/v1/employees/").Handler(createReverseProxy("tickets"))
	r.PathPrefix("/api/v1/stats/").Handler(createReverseProxy("tickets"))
	r.PathPrefix("/api/v1/counters/").Handler(createReverseProxy("counters"))
	r.PathPrefix("/api/v1/notifications/").Handler(createReverseProxy("notifications"))

	// Root endpoint
	r.HandleFunc("/", rootHandler).Methods("GET")

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API Gateway starting on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Gateway stopped")
}

// Handlers
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":  "Vodokanal API Gateway",
		"version":  "1.0.0",
		"status":   "running",
		"endpoints": map[string]interface{}{
			"auth":        "/api/v1/auth/*",
			"subscribers": "/api/v1/subscribers/*",
			"readings":    "/api/v1/readings/*",
			"billing":     "/api/v1/bills/*, /api/v1/tariffs/*",
			"payments":    "/api/v1/payments/*",
			"tickets":     "/api/v1/tickets/*",
			"counters":    "/api/v1/counters/*",
		},
		"health": "/health",
	})
}

func gatewayHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "gateway",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func servicesHealthCheck(w http.ResponseWriter, r *http.Request) {
	results := make([]ServiceHealth, 0, len(services))

	for name, service := range services {
		serviceURL := fmt.Sprintf("http://%s:%d/health", service.Host, service.Port)
		status := checkServiceHealth(serviceURL)

		results = append(results, ServiceHealth{
			Name:   name,
			Status: status,
			URL:    serviceURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  results,
	})
}

func servicesStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"gateway": map[string]interface{}{
			"status": "running",
			"port":   8080,
		},
	}

	for name, service := range services {
		serviceURL := fmt.Sprintf("http://%s:%d/health", service.Host, service.Port)
		healthStatus := checkServiceHealth(serviceURL)

		status[name] = map[string]interface{}{
			"status": healthStatus,
			"host":   service.Host,
			"port":   service.Port,
			"url":    fmt.Sprintf("http://%s:%d", service.Host, service.Port),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper functions
func createReverseProxy(serviceName string) http.Handler {
	service, exists := services[serviceName]
	if !exists {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Service not found", http.StatusNotFound)
		})
	}

	targetURL, _ := url.Parse(fmt.Sprintf("http://%s:%d", service.Host, service.Port))

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for %s: %v", serviceName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "service_unavailable",
			"message": fmt.Sprintf("Service %s is unavailable", serviceName),
			"service": serviceName,
		})
	}

	// Modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Add custom headers to indicate the request comes from gateway
		req.Header.Set("X-Gateway-Service", serviceName)
		req.Header.Set("X-Forwarded-By", "vodokanal-gateway")

		// Log the proxy request
		log.Printf("[GATEWAY] Proxying %s %s to %s (http://%s:%d%s)",
			req.Method, req.URL.Path, serviceName, service.Host, service.Port, req.URL.Path)
	}

	return proxy
}

func checkServiceHealth(healthURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return "error"
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "healthy"
	}

	return "unhealthy"
}

func forwardRequest(serviceName string, w http.ResponseWriter, r *http.Request) {
	service, exists := services[serviceName]
	if !exists {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	// Build target URL
	targetURL := fmt.Sprintf("http://%s:%d%s", service.Host, service.Port, r.URL.Path)

	// Copy the request body
	var body io.Reader = r.Body
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		body = strings.NewReader(string(bodyBytes))
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Add gateway headers
	proxyReq.Header.Set("X-Gateway-Service", serviceName)
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Error forwarding to %s: %v", serviceName, err)
		http.Error(w, "Service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)

	log.Printf("[GATEWAY] Forwarded %s %s to %s -> %d",
		r.Method, r.URL.Path, serviceName, resp.StatusCode)
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[GATEWAY] %s %s from %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("[GATEWAY] %s %s completed in %v", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

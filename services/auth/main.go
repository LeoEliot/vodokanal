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
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`

	// Subscriber information (if user is a subscriber)
	SubscriberID   string    `json:"subscriber_id,omitempty"`
	AccountNumber  string    `json:"account_number,omitempty"`

	// User profile
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Patronymic     string    `json:"patronymic,omitempty"`
	Phone          string    `json:"phone,omitempty"`

	// Role and permissions
	Role           string    `json:"role"`           // subscriber, employee, admin
	IsActive       bool      `json:"is_active"`

	// Timestamps
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Patronymic    string `json:"patronymic,omitempty"`
	Phone         string `json:"phone,omitempty"`
	SubscriberID  string `json:"subscriber_id,omitempty"`
	AccountNumber string `json:"account_number,omitempty"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Claims represents JWT claims
type Claims struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	SubscriberID string `json:"subscriber_id,omitempty"`
	jwt.RegisteredClaims
}

var (
	usersDB       = make(map[string]User)
	refreshTokens = make(map[string]string) // token -> userID
	dbMutex       sync.RWMutex
	jwtSecret     = []byte(getEnv("JWT_SECRET", "your-secret-key-change-in-production"))
	tokenExpiry   = getEnvDuration("TOKEN_EXPIRY", 24*time.Hour)
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func init() {
	// Create default admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	adminUser := User{
		ID:           "user_admin",
		Email:        "admin@vodokanal.ru",
		PasswordHash: string(hashedPassword),
		FirstName:    "Админ",
		LastName:     "Системный",
		Role:         "admin",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	usersDB[adminUser.ID] = adminUser

	// Create test subscriber
	hashedPassword2, _ := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
	testUser := User{
		ID:            "user_test",
		Email:         "test@example.com",
		PasswordHash:  string(hashedPassword2),
		FirstName:     "Иван",
		LastName:      "Иванов",
		Patronymic:    "Иванович",
		Phone:         "+7-999-123-45-67",
		SubscriberID:  "sub_test",
		AccountNumber: "000123456",
		Role:          "subscriber",
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	usersDB[testUser.ID] = testUser
}

func main() {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API Routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Auth endpoints (public)
	api.HandleFunc("/auth/register", register).Methods("POST")
	api.HandleFunc("/auth/login", login).Methods("POST")
	api.HandleFunc("/auth/refresh", refreshToken).Methods("POST")
	api.HandleFunc("/auth/logout", logout).Methods("POST")
	api.HandleFunc("/auth/verify", verifyToken).Methods("GET")

	// User endpoints (protected)
	api.HandleFunc("/users", requireAuth(getUsers)).Methods("GET")
	api.HandleFunc("/users/{id}", requireAuth(getUser)).Methods("GET")
	api.HandleFunc("/users/{id}", requireAuth(updateUser)).Methods("PUT")
	api.HandleFunc("/users/{id}", requireAuth(deleteUser)).Methods("DELETE")
	api.HandleFunc("/users/me", requireAuth(getCurrentUser)).Methods("GET")
	api.HandleFunc("/users/me/password", requireAuth(changePassword)).Methods("PUT")

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Auth Service starting on port 8081")
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
		"service":   "auth",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Auth handlers
func register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check if email already exists
	for _, user := range usersDB {
		if strings.ToLower(user.Email) == strings.ToLower(req.Email) {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create user
	user := User{
		ID:            fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Patronymic:    req.Patronymic,
		Phone:         req.Phone,
		SubscriberID:  req.SubscriberID,
		AccountNumber: req.AccountNumber,
		Role:          "subscriber", // Default role
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	usersDB[user.ID] = user

	// Generate token
	token, expiresAt, err := generateJWT(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	dbMutex.RLock()
	var user *User
	for _, u := range usersDB {
		if strings.ToLower(u.Email) == strings.ToLower(req.Email) {
			u := u
			user = &u
			break
		}
	}
	dbMutex.RUnlock()

	if user == nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.IsActive {
		http.Error(w, "Account is inactive", http.StatusForbidden)
		return
	}

	// Update last login
	dbMutex.Lock()
	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	usersDB[user.ID] = *user
	dbMutex.Unlock()

	// Generate token
	token, expiresAt, err := generateJWT(*user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Generate refresh token
	refreshToken := fmt.Sprintf("refresh_%d", time.Now().UnixNano())
	dbMutex.Lock()
	refreshTokens[refreshToken] = user.ID
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	})
}

func refreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.RLock()
	userID, exists := refreshTokens[req.RefreshToken]
	user, userExists := usersDB[userID]
	dbMutex.RUnlock()

	if !exists || !userExists {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if !user.IsActive {
		http.Error(w, "Account is inactive", http.StatusForbidden)
		return
	}

	// Generate new tokens
	token, expiresAt, err := generateJWT(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	newRefreshToken := fmt.Sprintf("refresh_%d", time.Now().UnixNano())

	dbMutex.Lock()
	delete(refreshTokens, req.RefreshToken)
	refreshTokens[newRefreshToken] = user.ID
	dbMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:        token,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User:         user,
	})
}

func logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	delete(refreshTokens, req.RefreshToken)
	dbMutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func verifyToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := validateJWT(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	dbMutex.RLock()
	user, exists := usersDB[claims.UserID]
	dbMutex.RUnlock()

	if !exists || !user.IsActive {
		http.Error(w, "User not found or inactive", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"user":  user,
	})
}

// User handlers
func getUsers(w http.ResponseWriter, r *http.Request, claims *Claims) {
	// Only admins can list all users
	if claims.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	users := make([]User, 0, len(usersDB))
	for _, u := range usersDB {
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request, claims *Claims) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Users can only view their own profile unless they're admin
	if claims.Role != "admin" && claims.UserID != id {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dbMutex.RLock()
	user, exists := usersDB[id]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getCurrentUser(w http.ResponseWriter, r *http.Request, claims *Claims) {
	dbMutex.RLock()
	user, exists := usersDB[claims.UserID]
	dbMutex.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request, claims *Claims) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Users can only update their own profile unless they're admin
	if claims.Role != "admin" && claims.UserID != id {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var updatedUser User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	user, exists := usersDB[id]
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update allowed fields
	if updatedUser.FirstName != "" {
		user.FirstName = updatedUser.FirstName
	}
	if updatedUser.LastName != "" {
		user.LastName = updatedUser.LastName
	}
	if updatedUser.Patronymic != "" {
		user.Patronymic = updatedUser.Patronymic
	}
	if updatedUser.Phone != "" {
		user.Phone = updatedUser.Phone
	}

	user.UpdatedAt = time.Now()
	usersDB[id] = user

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request, claims *Claims) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Only admins can delete users
	if claims.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	if _, exists := usersDB[id]; !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	delete(usersDB, id)
	w.WriteHeader(http.StatusNoContent)
}

func changePassword(w http.ResponseWriter, r *http.Request, claims *Claims) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "Old and new passwords are required", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		http.Error(w, "New password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	user, exists := usersDB[claims.UserID]
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		http.Error(w, "Invalid old password", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()
	usersDB[claims.UserID] = user

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func generateJWT(user User) (string, time.Time, error) {
	expiresAt := time.Now().Add(tokenExpiry)

	claims := Claims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		SubscriberID: user.SubscriberID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func validateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Middleware
func requireAuth(next func(http.ResponseWriter, *http.Request, *Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := validateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if user is still active
		dbMutex.RLock()
		user, exists := usersDB[claims.UserID]
		dbMutex.RUnlock()

		if !exists || !user.IsActive {
			http.Error(w, "User not found or inactive", http.StatusUnauthorized)
			return
		}

		next(w, r, claims)
	}
}

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

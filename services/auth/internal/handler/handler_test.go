package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/vodokanal/auth/internal/jwt"
	"github.com/vodokanal/auth/internal/model"
)

func setupTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

func TestHealthCheck(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestRegisterValidation(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	tests := []struct {
		name       string
		payload    map[string]interface{}
		expectCode int
	}{
		{
			name: "missing email",
			payload: map[string]interface{}{
				"password": "SecurePass123",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "not-an-email",
				"password": "SecurePass123",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "password too short",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "short",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: map[string]interface{}{
				"email": "test@example.com",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestAuthRequired(t *testing.T) {
	jwtMgr := jwt.NewManager("test-secret", 0, 0)
	h := &Handler{jwtMgr: jwtMgr}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequiredInvalidFormat(t *testing.T) {
	jwtMgr := jwt.NewManager("test-secret", 0, 0)
	h := &Handler{jwtMgr: jwtMgr}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequiredInvalidToken(t *testing.T) {
	jwtMgr := jwt.NewManager("test-secret", 0, 0)
	h := &Handler{jwtMgr: jwtMgr}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginValidation(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	tests := []struct {
		name       string
		payload    map[string]interface{}
		expectCode int
	}{
		{
			name: "missing email",
			payload: map[string]interface{}{
				"password": "SecurePass123",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: map[string]interface{}{
				"email": "test@example.com",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "not-an-email",
				"password": "SecurePass123",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestRefreshTokenValidation(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer([]byte("{}")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Benchmark JSON marshal for login
func BenchmarkLoginRequestMarshal(b *testing.B) {
	req := model.LoginRequest{
		Email:    "user@example.com",
		Password: "SecurePass123",
	}

	for i := 0; i < b.N; i++ {
		json.Marshal(req)
	}
}

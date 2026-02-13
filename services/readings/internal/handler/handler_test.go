package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/vodokanal/readings/internal/model"
)

// MockService for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) Create(ctx context.Context, req *model.CreateReadingRequest, submittedBy *int) (*model.Reading, error) {
	args := m.Called(ctx, req, submittedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reading), args.Error(1)
}

func (m *MockService) Get(ctx context.Context, id int) (*model.Reading, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reading), args.Error(1)
}

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

func TestCreateReadingValidation(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	tests := []struct {
		name       string
		payload    map[string]interface{}
		expectCode int
	}{
		{
			name: "missing subscriber_id",
			payload: map[string]interface{}{
				"counter_id": 1,
				"value":      123.45,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing counter_id",
			payload: map[string]interface{}{
				"subscriber_id": 1,
				"value":         123.45,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing value",
			payload: map[string]interface{}{
				"subscriber_id": 1,
				"counter_id":    1,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "negative value",
			payload: map[string]interface{}{
				"subscriber_id": 1,
				"counter_id":    1,
				"value":         -10.0,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "zero value",
			payload: map[string]interface{}{
				"subscriber_id": 1,
				"counter_id":    1,
				"value":         0,
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/readings", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestGetReadingInvalidID(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/readings/invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBySubscriberInvalidID(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscribers/invalid/readings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetByCounterInvalidID(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/counters/invalid/readings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStatisticsMissingParams(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/readings/statistics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateReadingValidation(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	tests := []struct {
		name       string
		url        string
		payload    map[string]interface{}
		expectCode int
	}{
		{
			name: "verify without verified_by",
			url:  "/api/v1/readings/1",
			payload: map[string]interface{}{
				"verified": true,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "negative value",
			url:  "/api/v1/readings/1",
			payload: map[string]interface{}{
				"value": -10.0,
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPut, tt.url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestVerifyWithoutAuth(t *testing.T) {
	h := &Handler{}
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/readings/1/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

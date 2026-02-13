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
	"github.com/stretchr/testify/mock"
	"github.com/vodokanal/subscribers/internal/model"
	"github.com/vodokanal/subscribers/internal/service"
)

// MockService for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) List(ctx context.Context, opts service.ListOptions) (*service.ListResult, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ListResult), args.Error(1)
}

func (m *MockService) Get(ctx context.Context, id int) (*model.Subscriber, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func (m *MockService) GetByAccountNumber(ctx context.Context, accountNumber string) (*model.Subscriber, error) {
	args := m.Called(ctx, accountNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func (m *MockService) GetByEmail(ctx context.Context, email string) (*model.Subscriber, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func (m *MockService) Create(ctx context.Context, req *model.CreateSubscriberRequest) (*model.Subscriber, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func (m *MockService) Update(ctx context.Context, id int, req *model.UpdateSubscriberRequest) (*model.Subscriber, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func (m *MockService) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.RegisterRoutes(r)
	return r
}

func TestHealthCheck(t *testing.T) {
	mockSvc := new(MockService)
	h := New(mockSvc)
	r := setupTestRouter(h)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestListSubscribers(t *testing.T) {
	mockSvc := new(MockService)
	h := New(mockSvc)
	r := setupTestRouter(h)

	expected := &service.ListResult{
		Data: []model.Subscriber{
			{ID: 1, AccountNumber: "12345", LastName: "Иванов", FirstName: "Иван"},
		},
		Page:  1,
		Limit: 20,
		Total: 1,
	}

	mockSvc.On("List", mock.Anything, mock.Anything).Return(expected, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscribers?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response service.ListResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.Total)
}

func TestCreateSubscriber(t *testing.T) {
	mockSvc := new(MockService)
	h := New(mockSvc)
	r := setupTestRouter(h)

	reqBody := model.CreateSubscriberRequest{
		AccountNumber: "12345",
		LastName:      "Иванов",
		FirstName:     "Иван",
		MiddleName:    "Иванович",
		Email:         "ivanov@example.com",
		Phone:         "+79991234567",
		Address:       "г. Москва",
	}

	expected := &model.Subscriber{
		ID:            1,
		AccountNumber: "12345",
		LastName:      "Иванов",
		FirstName:     "Иван",
		Email:         "ivanov@example.com",
		Phone:         "+79991234567",
		Address:       "г. Москва",
	}

	mockSvc.On("Create", mock.Anything, &reqBody).Return(expected, nil)

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscribers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response model.Subscriber
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expected.AccountNumber, response.AccountNumber)
}

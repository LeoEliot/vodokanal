package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/vodokanal/counters/internal/model"
	"github.com/vodokanal/counters/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	logger  *zap.Logger
}

func New(svc *service.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: svc,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		counters := api.Group("/counters")
		{
			counters.GET("", h.List)
			counters.POST("", h.Create)
			counters.GET("/dashboard", h.GetDashboard)
			counters.GET("/:id", h.Get)
			counters.PUT("/:id", h.Update)
			counters.DELETE("/:id", h.Delete)
			counters.POST("/:id/activate", h.Activate)
			counters.POST("/:id/deactivate", h.Deactivate)
			counters.POST("/:id/verify", h.Verify)
			counters.GET("/:id/status", h.GetStatus)
		}

		// Subscriber scoped endpoints
		subscribers := api.Group("/subscribers/:subscriber_id")
		{
			subscribers.GET("/counters", h.GetBySubscriber)
			subscribers.GET("/counters/active", h.GetActiveBySubscriber)
			subscribers.GET("/counters/statuses", h.GetStatusesBySubscriber)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "counters",
		})
	})
}

// Create creates a new counter
func (h *Handler) Create(c *gin.Context) {
	var req model.CreateCounterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	counter, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrDuplicateSerialNumber) {
			c.JSON(http.StatusConflict, gin.H{"error": "counter with this serial number already exists"})
			return
		}
		h.logger.Error("Failed to create counter", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, counter)
}

// Get retrieves a counter by ID
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	counter, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, counter)
}

// List retrieves counters with pagination and filters
func (h *Handler) List(c *gin.Context) {
	subscriberID, _ := strconv.Atoi(c.Query("subscriber_id"))
	counterType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var subID *int
	if subscriberID > 0 {
		subID = &subscriberID
	}

	var cntType *string
	if counterType != "" {
		cntType = &counterType
	}

	result, err := h.service.List(c.Request.Context(), subID, cntType, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBySubscriber retrieves counters for a subscriber
func (h *Handler) GetBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	counters, err := h.service.GetBySubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": counters})
}

// GetActiveBySubscriber retrieves active counters for a subscriber
func (h *Handler) GetActiveBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	counters, err := h.service.GetActiveBySubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": counters})
}

// GetStatusesBySubscriber retrieves counter statuses for a subscriber
func (h *Handler) GetStatusesBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	statuses, err := h.service.GetStatusesBySubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": statuses})
}

// Update updates a counter
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req model.UpdateCounterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	counter, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, model.ErrDuplicateSerialNumber) {
			c.JSON(http.StatusConflict, gin.H{"error": "counter with this serial number already exists"})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, counter)
}

// Delete deletes a counter
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Activate activates a counter
func (h *Handler) Activate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Activate(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "counter activated"})
}

// Deactivate deactivates a counter
func (h *Handler) Deactivate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Deactivate(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "counter deactivated"})
}

// Verify verifies a counter
func (h *Handler) Verify(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	verifiedBy, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	var req model.VerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	counter, err := h.service.Verify(c.Request.Context(), id, verifiedBy, &req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, counter)
}

// GetStatus retrieves counter status
func (h *Handler) GetStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	status, err := h.service.GetStatus(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetDashboard retrieves dashboard statistics
func (h *Handler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/vodokanal/readings/internal/model"
	"github.com/vodokanal/readings/internal/service"
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
		readings := api.Group("/readings")
		{
			readings.GET("", h.List)
			readings.POST("", h.Create)
			readings.GET("/latest", h.GetLatest)
			readings.GET("/monthly", h.GetMonthly)
			readings.GET("/statistics", h.GetStatistics)
			readings.GET("/:id", h.Get)
			readings.PUT("/:id", h.Update)
			readings.DELETE("/:id", h.Delete)
			readings.POST("/:id/verify", h.Verify)
		}

		// Subscriber scoped endpoints
		subscribers := api.Group("/subscribers/:subscriber_id")
		{
			subscribers.GET("/readings", h.GetBySubscriber)
			subscribers.GET("/readings/latest", h.GetLatestBySubscriber)
			subscribers.GET("/readings/monthly", h.GetMonthlyBySubscriber)
		}

		// Counter scoped endpoints
		counters := api.Group("/counters/:counter_id")
		{
			counters.GET("/readings", h.GetByCounter)
			counters.GET("/readings/latest", h.GetLatestByCounter)
			counters.GET("/readings/statistics", h.GetStatisticsByCounter)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "readings",
		})
	})
}

// Create creates a new reading
func (h *Handler) Create(c *gin.Context) {
	var req model.CreateReadingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context if authenticated
	var submittedBy *int
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(int); ok {
			submittedBy = &uid
		}
	}

	reading, err := h.service.Create(c.Request.Context(), &req, submittedBy)
	if err != nil {
		h.logger.Error("Failed to create reading", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reading)
}

// Get retrieves a reading by ID
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	reading, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "reading not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// List retrieves readings with pagination and filters
func (h *Handler) List(c *gin.Context) {
	subscriberID, _ := strconv.Atoi(c.Query("subscriber_id"))
	counterID, _ := strconv.Atoi(c.Query("counter_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var subID, cntID *int
	if subscriberID > 0 {
		subID = &subscriberID
	}
	if counterID > 0 {
		cntID = &counterID
	}

	result, err := h.service.List(c.Request.Context(), subID, cntID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetLatest retrieves latest readings
func (h *Handler) GetLatest(c *gin.Context) {
	subscriberID, _ := strconv.Atoi(c.Query("subscriber_id"))
	counterID, _ := strconv.Atoi(c.Query("counter_id"))

	var subID, cntID *int
	if subscriberID > 0 {
		subID = &subscriberID
	}
	if counterID > 0 {
		cntID = &counterID
	}

	result, err := h.service.List(c.Request.Context(), subID, cntID, 1, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBySubscriber retrieves readings for a subscriber
func (h *Handler) GetBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	subID := &subscriberID

	result, err := h.service.List(c.Request.Context(), subID, nil, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByCounter retrieves readings for a counter
func (h *Handler) GetByCounter(c *gin.Context) {
	counterID, err := strconv.Atoi(c.Param("counter_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid counter_id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	cntID := &counterID

	result, err := h.service.List(c.Request.Context(), nil, cntID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetLatestBySubscriber retrieves latest readings for a subscriber
func (h *Handler) GetLatestBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	readings, err := h.service.GetLatestBySubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": readings})
}

// GetLatestByCounter retrieves latest reading for a counter
func (h *Handler) GetLatestByCounter(c *gin.Context) {
	counterID, err := strconv.Atoi(c.Param("counter_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid counter_id"})
		return
	}

	reading, err := h.service.GetLatestByCounter(c.Request.Context(), counterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no readings found for this counter"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// GetMonthly retrieves monthly readings
func (h *Handler) GetMonthly(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Query("subscriber_id"))
	if err != nil || subscriberID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriber_id is required"})
		return
	}

	months, _ := strconv.Atoi(c.DefaultQuery("months", "12"))

	readings, err := h.service.GetMonthlyReadings(c.Request.Context(), subscriberID, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": readings})
}

// GetMonthlyBySubscriber retrieves monthly readings for a subscriber
func (h *Handler) GetMonthlyBySubscriber(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	months, _ := strconv.Atoi(c.DefaultQuery("months", "12"))

	readings, err := h.service.GetMonthlyReadings(c.Request.Context(), subscriberID, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": readings})
}

// GetStatistics retrieves consumption statistics
func (h *Handler) GetStatistics(c *gin.Context) {
	subscriberID, _ := strconv.Atoi(c.Query("subscriber_id"))
	counterID, _ := strconv.Atoi(c.Query("counter_id"))

	if subscriberID == 0 || counterID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriber_id and counter_id are required"})
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), subscriberID, counterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStatisticsByCounter retrieves statistics for a counter
func (h *Handler) GetStatisticsByCounter(c *gin.Context) {
	counterID, err := strconv.Atoi(c.Param("counter_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid counter_id"})
		return
	}

	// Get subscriber_id from query or get it from counter
	subscriberID, _ := strconv.Atoi(c.Query("subscriber_id"))
	if subscriberID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriber_id is required"})
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), subscriberID, counterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Update updates a reading
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req model.UpdateReadingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	reading, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "reading not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// Delete deletes a reading
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "reading not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Verify verifies a reading
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

	reading, err := h.service.Verify(c.Request.Context(), id, verifiedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "reading not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// GetByDateRange retrieves readings within a date range
func (h *Handler) GetByDateRange(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("subscriber_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriber_id"})
		return
	}

	startDate := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	readings, err := h.service.GetByDateRange(c.Request.Context(), subscriberID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": readings})
}

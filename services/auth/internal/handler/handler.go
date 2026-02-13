package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vodokanal/auth/internal/jwt"
	"github.com/vodokanal/auth/internal/model"
	"github.com/vodokanal/auth/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	jwtMgr  *jwt.Manager
	logger  *zap.Logger
}

func New(svc *service.Service, jwtMgr *jwt.Manager, logger *zap.Logger) *Handler {
	return &Handler{
		service: svc,
		jwtMgr:  jwtMgr,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
			auth.POST("/logout", h.AuthRequired(), h.Logout)
			auth.POST("/logout-all", h.AuthRequired(), h.LogoutAll)

			// Password management
			auth.POST("/change-password", h.AuthRequired(), h.ChangePassword)
		}

		// User endpoints
		users := api.Group("/users")
		{
			users.GET("/me", h.AuthRequired(), h.GetCurrentUser)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth",
		})
	})
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.service.Register(c.Request.Context(), req.Email, req.Password, nil)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
			return
		}
		h.logger.Error("Registration failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user account is inactive"})
			return
		}
		h.logger.Error("Login failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, jwt.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		if errors.Is(err, service.ErrTokenExpired) || errors.Is(err, jwt.ErrExpiredToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has expired"})
			return
		}
		h.logger.Error("Token refresh failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	userID := c.GetInt("user_id")
	refreshToken := c.GetHeader("X-Refresh-Token")

	_ = h.service.Logout(c.Request.Context(), userID, refreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// LogoutAll handles user logout from all devices
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := c.GetInt("user_id")

	if err := h.service.LogoutAll(c.Request.Context(), userID); err != nil {
		h.logger.Error("Logout all failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out from all devices"})
}

// ChangePassword handles password change
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// GetCurrentUser returns the current user
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := c.GetInt("user_id")

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// AuthRequired is a middleware that checks for valid JWT
func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := h.jwtMgr.ValidateAccessToken(parts[1])
		if err != nil {
			if errors.Is(err, jwt.ErrExpiredToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireAdmin is a middleware that checks for admin role
func (h *Handler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.AuthRequired()(c)

		role := c.GetString("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}

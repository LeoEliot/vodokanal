package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Auth service proxy
		auth := v1.Group("/auth")
		{
			auth.POST("/register", proxyTo("http://auth:8081"))
			auth.POST("/login", proxyTo("http://auth:8081"))
			auth.POST("/refresh", proxyTo("http://auth:8081"))
		}

		// Subscribers service proxy
		subs := v1.Group("/subscribers")
		{
			subs.GET("", proxyTo("http://subscribers:8082"))
			subs.GET("/:id", proxyTo("http://subscribers:8082"))
			subs.PUT("/:id", proxyTo("http://subscribers:8082"))
		}

		// Readings service proxy
		readings := v1.Group("/readings")
		{
			readings.POST("", proxyTo("http://readings:8083"))
			readings.GET("/:subscriber_id", proxyTo("http://readings:8083"))
		}

		// Billing service proxy
		billing := v1.Group("/billing")
		{
			billing.GET("/invoices", proxyTo("http://billing:8084"))
			billing.GET("/invoices/:id", proxyTo("http://billing:8084"))
		}

		// Payments service proxy
		payments := v1.Group("/payments")
		{
			payments.POST("", proxyTo("http://payments:8085"))
			payments.GET("/:id", proxyTo("http://payments:8085"))
		}

		// Tickets service proxy
		tickets := v1.Group("/tickets")
		{
			tickets.POST("", proxyTo("http://tickets:8086"))
			tickets.GET("", proxyTo("http://tickets:8086"))
		}

		// Counters service proxy
		counters := v1.Group("/counters")
		{
			counters.GET("", proxyTo("http://counters:8087"))
			counters.POST("", proxyTo("http://counters:8087"))
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", getEnv("PORT", "8080")),
		Handler: r,
	}

	go func() {
		logger.Info("Starting gateway server", zap.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Server exited")
}

func proxyTo(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"message": "Proxy to " + target})
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

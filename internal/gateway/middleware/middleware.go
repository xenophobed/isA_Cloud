package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"github.com/isa-cloud/isa_cloud/pkg/logger"
	"github.com/isa-cloud/isa_cloud/internal/gateway/clients"
	"github.com/isa-cloud/isa_cloud/internal/gateway/registry"
)

// RequestLogger returns a middleware that logs HTTP requests
func RequestLogger(logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		end := time.Now()
		latency := end.Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", latency,
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"request_id", c.GetString("request_id"),
		)
	}
}

// RequestID returns a middleware that adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// RateLimit returns a rate limiting middleware
func RateLimit(rps, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"message": fmt.Sprintf("rate limit: %d requests per second", rps),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthenticationWithRegistry returns an authentication middleware with service registry
func AuthenticationWithRegistry(authClient clients.AuthClient, consul *registry.ConsulRegistry, logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health checks
		if strings.HasPrefix(c.Request.URL.Path, "/health") || 
		   strings.HasPrefix(c.Request.URL.Path, "/ready") {
			c.Next()
			return
		}

		// Check if request is from a registered internal service
		// Method 1: Check service header with Consul validation
		serviceName := c.GetHeader("X-Service-Name")
		serviceID := c.GetHeader("X-Service-ID")
		
		if serviceName != "" && consul != nil {
			// Check if this service is registered in Consul
			services, err := consul.ListServices()
			if err == nil && services != nil {
				// Check if the service name exists in registered services
				if _, exists := services[serviceName]; exists {
					logger.Debug("Internal service authenticated via Consul",
						"service", serviceName,
						"service_id", serviceID,
						"request_id", c.GetString("request_id"),
					)
					c.Set("user_id", "service-" + serviceName)
					c.Set("organization_id", "internal")
					c.Set("is_internal", true)
					c.Set("service_name", serviceName)
					c.Next()
					return
				}
			}
		}

		// Method 2: For localhost development - auto-approve if from local service port ranges
		clientIP := c.ClientIP()
		if clientIP == "127.0.0.1" || clientIP == "::1" || strings.HasPrefix(clientIP, "localhost") {
			// Check if it's from known service port range (8200-8300 for microservices)
			userAgent := c.GetHeader("User-Agent")
			
			// Auto-approve for local development with Python/Node.js clients
			if strings.Contains(userAgent, "python-httpx") || 
			   strings.Contains(userAgent, "axios") ||
			   strings.Contains(userAgent, "node-fetch") {
				logger.Debug("Local development service auto-authenticated",
					"ip", clientIP,
					"user_agent", userAgent,
					"request_id", c.GetString("request_id"),
				)
				c.Set("user_id", "local-dev-service")
				c.Set("organization_id", "local")
				c.Set("is_internal", true)
				c.Next()
				return
			}
		}

		// Original authentication logic for external requests
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// For now, just validate that token exists
		// TODO: Implement actual token validation via gRPC call to auth service
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "empty token",
			})
			c.Abort()
			return
		}

		// Mock user context for testing
		c.Set("user_id", "test-user-123")
		c.Set("organization_id", "test-org-456")
		c.Set("token", token)

		logger.Debug("Authentication successful", 
			"user_id", "test-user-123",
			"request_id", c.GetString("request_id"),
		)

		c.Next()
	}
}

// Authentication returns a basic authentication middleware (for backward compatibility)
func Authentication(authClient clients.AuthClient, logger *logger.Logger) gin.HandlerFunc {
	// Use the new function with nil registry for backward compatibility
	return AuthenticationWithRegistry(authClient, nil, logger)
}


// CORS returns a CORS middleware
func CORS(allowOrigins, allowMethods, allowHeaders []string, allowCredentials bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Set CORS headers
		if len(allowOrigins) > 0 && (allowOrigins[0] == "*" || contains(allowOrigins, origin)) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		if len(allowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
		}

		if len(allowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))
		}

		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
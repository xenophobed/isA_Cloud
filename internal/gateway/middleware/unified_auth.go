package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isa-cloud/isa_cloud/internal/gateway/clients"
	"github.com/isa-cloud/isa_cloud/internal/gateway/registry"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// AuthService response structs
type TokenVerificationResponse struct {
	Valid     bool   `json:"valid"`
	Provider  string `json:"provider"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expires_at"`
	Error     string `json:"error"`
}

type APIKeyVerificationResponse struct {
	Valid          bool     `json:"valid"`
	KeyID          string   `json:"key_id"`
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Permissions    []string `json:"permissions"`
	CreatedAt      string   `json:"created_at"`
	LastUsed       string   `json:"last_used"`
	Error          string   `json:"error"`
}

// AuthorizationService response structs
type AccessCheckResponse struct {
	HasAccess          bool   `json:"has_access"`
	UserAccessLevel    string `json:"user_access_level"`
	PermissionSource   string `json:"permission_source"`
	SubscriptionTier   string `json:"subscription_tier"`
	OrganizationPlan   string `json:"organization_plan"`
	Reason             string `json:"reason"`
	ExpiresAt          string `json:"expires_at"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// UnifiedAuthentication provides a unified authentication middleware that:
// 1. Routes external requests through Auth Service (8202)
// 2. Maintains compatibility with service-specific auth (Agent, MCP)
// 3. Handles internal service-to-service communication
func UnifiedAuthentication(authClient clients.AuthClient, consul *registry.ConsulRegistry, logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health checks and public endpoints
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Check for internal service authentication first
		if authenticated := handleInternalServiceAuth(c, consul, logger); authenticated {
			return
		}

		// Handle external authentication via Auth Service
		if authenticated := handleExternalAuth(c, authClient, logger); authenticated {
			return
		}

		// Authentication failed
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
			"message": "valid JWT token or API key required",
		})
		c.Abort()
	}
}

// isPublicEndpoint checks if the endpoint should bypass authentication
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/ready",
		"/api/v1/gateway/services", // Allow service discovery
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}
	return false
}

// handleInternalServiceAuth handles service-to-service authentication
func handleInternalServiceAuth(c *gin.Context, consul *registry.ConsulRegistry, logger *logger.Logger) bool {
	// Method 1: Explicit internal service header with Consul validation
	serviceName := c.GetHeader("X-Service-Name")
	serviceSecret := c.GetHeader("X-Service-Secret")
	
	if serviceName != "" && serviceSecret != "" && consul != nil {
		// Validate service is registered in Consul
		if isValidInternalService(serviceName, consul, logger) {
			// In production, validate serviceSecret against a shared secret
			// For now, accept any registered service
			logger.Debug("Internal service authenticated",
				"service", serviceName,
				"path", c.Request.URL.Path,
			)
			c.Set("user_id", "service-"+serviceName)
			c.Set("organization_id", "internal")
			c.Set("is_internal", true)
			c.Set("service_name", serviceName)
			c.Next()
			return true
		}
	}

	// Method 2: Development mode - localhost with service user agents
	clientIP := c.ClientIP()
	if isLocalhost(clientIP) {
		userAgent := c.GetHeader("User-Agent")
		if isServiceUserAgent(userAgent) {
			logger.Debug("Local development service authenticated",
				"ip", clientIP,
				"user_agent", userAgent,
				"path", c.Request.URL.Path,
			)
			c.Set("user_id", "local-dev-service")
			c.Set("organization_id", "local")
			c.Set("is_internal", true)
			c.Next()
			return true
		}
	}

	return false
}

// handleExternalAuth handles external user authentication via Auth Service
func handleExternalAuth(c *gin.Context, authClient clients.AuthClient, logger *logger.Logger) bool {
	// Try JWT token authentication first
	if authenticated := handleJWTAuth(c, authClient, logger); authenticated {
		return true
	}

	// Try API key authentication
	if authenticated := handleAPIKeyAuth(c, authClient, logger); authenticated {
		return true
	}

	return false
}

// handleJWTAuth validates JWT tokens via Auth Service
func handleJWTAuth(c *gin.Context, authClient clients.AuthClient, logger *logger.Logger) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	// Parse Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}

	token := parts[1]
	if token == "" {
		return false
	}

	// Call Auth Service to verify token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create verification request
	payload := map[string]interface{}{
		"token": token,
	}

	response, err := makeAuthServiceRequest(ctx, "http://localhost:8202/api/v1/auth/verify-token", payload)
	if err != nil {
		logger.Error("Auth service request failed", "error", err)
		return false
	}

	var tokenResp TokenVerificationResponse
	if err := json.Unmarshal(response, &tokenResp); err != nil {
		logger.Error("Failed to parse token verification response", "error", err)
		return false
	}

	if !tokenResp.Valid {
		logger.Debug("Token validation failed", "error", tokenResp.Error)
		return false
	}

	// Set user context
	c.Set("user_id", tokenResp.UserID)
	c.Set("email", tokenResp.Email)
	c.Set("provider", tokenResp.Provider)
	c.Set("is_internal", false)
	c.Set("auth_method", "jwt")

	// Check resource-specific permissions if needed
	if hasResourceAccess := checkResourcePermissions(c, tokenResp.UserID, logger); !hasResourceAccess {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions",
			"message": "user does not have permission to access this resource",
		})
		c.Abort()
		return false
	}

	logger.Debug("JWT authentication successful",
		"user_id", tokenResp.UserID,
		"provider", tokenResp.Provider,
	)

	c.Next()
	return true
}

// handleAPIKeyAuth validates API keys via Auth Service
func handleAPIKeyAuth(c *gin.Context, authClient clients.AuthClient, logger *logger.Logger) bool {
	// Check multiple sources for API key
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		apiKey = c.Query("api_key")
	}
	if apiKey == "" {
		if cookie, err := c.Cookie("api_key"); err == nil {
			apiKey = cookie
		}
	}

	if apiKey == "" {
		return false
	}

	// Call Auth Service to verify API key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"api_key": apiKey,
	}

	response, err := makeAuthServiceRequest(ctx, "http://localhost:8202/api/v1/auth/verify-api-key", payload)
	if err != nil {
		logger.Error("Auth service API key verification failed", "error", err)
		return false
	}

	var keyResp APIKeyVerificationResponse
	if err := json.Unmarshal(response, &keyResp); err != nil {
		logger.Error("Failed to parse API key verification response", "error", err)
		return false
	}

	if !keyResp.Valid {
		logger.Debug("API key validation failed", "error", keyResp.Error)
		return false
	}

	// Set user context
	c.Set("user_id", "api-key-"+keyResp.KeyID)
	c.Set("organization_id", keyResp.OrganizationID)
	c.Set("api_key_name", keyResp.Name)
	c.Set("permissions", keyResp.Permissions)
	c.Set("is_internal", false)
	c.Set("auth_method", "api_key")

	logger.Debug("API key authentication successful",
		"key_id", keyResp.KeyID,
		"organization_id", keyResp.OrganizationID,
		"name", keyResp.Name,
	)

	c.Next()
	return true
}

// Helper functions

func isValidInternalService(serviceName string, consul *registry.ConsulRegistry, logger *logger.Logger) bool {
	if consul == nil {
		return false
	}

	services, err := consul.ListServices()
	if err != nil {
		logger.Error("Failed to list Consul services", "error", err)
		return false
	}

	_, exists := services[serviceName]
	return exists
}

func isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "localhost")
}

func isServiceUserAgent(userAgent string) bool {
	serviceUserAgents := []string{
		"python-httpx",
		"axios",
		"node-fetch",
		"go-resty",
		"curl",
	}

	userAgent = strings.ToLower(userAgent)
	for _, serviceUA := range serviceUserAgents {
		if strings.Contains(userAgent, serviceUA) {
			return true
		}
	}
	return false
}

func makeAuthServiceRequest(ctx context.Context, url string, payload map[string]interface{}) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody := make([]byte, 0, resp.ContentLength)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			responseBody = append(responseBody, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth service returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// checkResourcePermissions validates user permissions for specific resources via Authorization Service
func checkResourcePermissions(c *gin.Context, userID string, logger *logger.Logger) bool {
	// Determine resource type and name from the request path
	resourceType, resourceName, requiredLevel := getResourceInfoFromPath(c.Request.URL.Path)
	
	// Skip permission check for public resources or if resource type is not recognized
	if resourceType == "" || requiredLevel == "" {
		return true
	}

	// Call Authorization Service to check access
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"user_id": userID,
		"resource_type": resourceType,
		"resource_name": resourceName,
		"required_access_level": requiredLevel,
	}

	response, err := makeAuthServiceRequest(ctx, "http://localhost:8203/api/v1/authorization/check-access", payload)
	if err != nil {
		logger.Error("Authorization service request failed", "error", err, "user_id", userID)
		// In case of service failure, allow access for now (fail-open policy)
		return true
	}

	var accessResp AccessCheckResponse
	if err := json.Unmarshal(response, &accessResp); err != nil {
		logger.Error("Failed to parse access check response", "error", err)
		return true
	}

	if !accessResp.HasAccess {
		logger.Debug("Access denied by authorization service",
			"user_id", userID,
			"resource_type", resourceType,
			"resource_name", resourceName,
			"reason", accessResp.Reason,
		)
		return false
	}

	// Store permission info in context for downstream services
	c.Set("access_level", accessResp.UserAccessLevel)
	c.Set("permission_source", accessResp.PermissionSource)
	c.Set("subscription_tier", accessResp.SubscriptionTier)

	logger.Debug("Access granted by authorization service",
		"user_id", userID,
		"resource_type", resourceType,
		"access_level", accessResp.UserAccessLevel,
		"permission_source", accessResp.PermissionSource,
	)

	return true
}

// getResourceInfoFromPath extracts resource information from the request path
func getResourceInfoFromPath(path string) (resourceType, resourceName, requiredLevel string) {
	// Define resource mappings based on API paths using Authorization Service's expected types
	if strings.HasPrefix(path, "/api/v1/blockchain/") {
		return "api_endpoint", "blockchain_" + extractBlockchainResource(path), "read_only"
	}
	
	if strings.HasPrefix(path, "/api/v1/agents/") {
		return "api_endpoint", "agent_chat", getAgentAccessLevel(path)
	}
	
	if strings.HasPrefix(path, "/api/v1/mcp/") {
		return "mcp_tool", extractMCPResource(path), getMCPAccessLevel(path)
	}
	
	if strings.HasPrefix(path, "/api/v1/gateway/") {
		return "api_endpoint", "gateway_management", "read_only"
	}

	// Unknown resource, skip permission check
	return "", "", ""
}

// Helper functions to extract resource details from specific API paths

func extractBlockchainResource(path string) string {
	if strings.Contains(path, "/balance/") {
		return "balance_check"
	}
	if strings.Contains(path, "/transaction") {
		return "transaction"
	}
	if strings.Contains(path, "/status") {
		return "status"
	}
	return "blockchain_general"
}

func getAgentAccessLevel(path string) string {
	if strings.Contains(path, "/api/chat") {
		return "read_write" // Chat requires read_write
	}
	return "read_only"
}

func extractMCPResource(path string) string {
	if strings.Contains(path, "/search") {
		return "search"
	}
	if strings.Contains(path, "/tools/call") {
		return "tool_execution"
	}
	if strings.Contains(path, "/prompts/get") {
		return "prompt_access"
	}
	return "mcp_general"
}

func getMCPAccessLevel(path string) string {
	if strings.Contains(path, "/tools/call") {
		return "read_write" // Tool execution requires read_write
	}
	return "read_only"
}
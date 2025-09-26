package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/isa-cloud/isa_cloud/internal/gateway/clients"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// BaseHandler contains common dependencies for all handlers
type BaseHandler struct {
	logger *logger.Logger
}

// UserHandler handles user-related requests
type UserHandler struct {
	BaseHandler
	client clients.UserClient
}

// AuthHandler handles authentication requests
type AuthHandler struct {
	BaseHandler
	client clients.AuthClient
}

// AgentHandler handles agent-related requests
type AgentHandler struct {
	BaseHandler
	client clients.AgentClient
}

// ModelHandler handles model-related requests
type ModelHandler struct {
	BaseHandler
	client clients.ModelClient
}

// MCPHandler handles MCP resource requests
type MCPHandler struct {
	BaseHandler
	client clients.MCPClient
}

// NewUserHandler creates a new user handler
func NewUserHandler(client clients.UserClient, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		BaseHandler: BaseHandler{logger: logger},
		client:      client,
	}
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(client clients.AuthClient, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		BaseHandler: BaseHandler{logger: logger},
		client:      client,
	}
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(client clients.AgentClient, logger *logger.Logger) *AgentHandler {
	return &AgentHandler{
		BaseHandler: BaseHandler{logger: logger},
		client:      client,
	}
}

// NewModelHandler creates a new model handler
func NewModelHandler(client clients.ModelClient, logger *logger.Logger) *ModelHandler {
	return &ModelHandler{
		BaseHandler: BaseHandler{logger: logger},
		client:      client,
	}
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(client clients.MCPClient, logger *logger.Logger) *MCPHandler {
	return &MCPHandler{
		BaseHandler: BaseHandler{logger: logger},
		client:      client,
	}
}

// User Handler Routes
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/me", h.getCurrentUser)
	router.GET("/:user_id", h.getUser)
}

func (h *UserHandler) getCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	user, err := h.client.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get current user", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (h *UserHandler) getUser(c *gin.Context) {
	userID := c.Param("user_id")
	
	user, err := h.client.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

// Auth Handler Routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/verify", h.verifyToken)
}

func (h *AuthHandler) verifyToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.client.VerifyToken(c.Request.Context(), req.Token)
	if err != nil {
		h.logger.Error("Failed to verify token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// Agent Handler Routes
func (h *AgentHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("", h.listAgents)
	router.GET("/:agent_id", h.getAgent)
}

func (h *AgentHandler) listAgents(c *gin.Context) {
	orgID := c.GetString("organization_id")
	
	agents, err := h.client.ListAgents(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to list agents", "error", err, "org_id", orgID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"agents":  agents.Agents,
		"total":   agents.Total,
	})
}

func (h *AgentHandler) getAgent(c *gin.Context) {
	agentID := c.Param("agent_id")
	
	// Mock response for now
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"agent": gin.H{
			"agent_id": agentID,
			"name":     "Mock Agent",
			"type":     "chat",
			"status":   "running",
		},
	})
}

// Model Handler Routes
func (h *ModelHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("", h.listModels)
	router.GET("/:model_id", h.getModel)
	router.POST("/:model_id/generate", h.generateText)
}

func (h *ModelHandler) listModels(c *gin.Context) {
	orgID := c.GetString("organization_id")
	
	models, err := h.client.ListModels(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to list models", "error", err, "org_id", orgID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"models":  models.Models,
		"total":   models.Total,
	})
}

func (h *ModelHandler) getModel(c *gin.Context) {
	modelID := c.Param("model_id")
	
	// Mock response for now
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"model": gin.H{
			"model_id": modelID,
			"name":     "Mock Model",
			"type":     "llm",
			"status":   "ready",
		},
	})
}

func (h *ModelHandler) generateText(c *gin.Context) {
	modelID := c.Param("model_id")
	
	var req struct {
		Prompt    string  `json:"prompt" binding:"required"`
		MaxTokens int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock response for now
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": gin.H{
			"model_id": modelID,
			"prompt":   req.Prompt,
			"text":     "This is a mock response from " + modelID,
			"tokens":   42,
		},
	})
}

// MCP Handler Routes
func (h *MCPHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/resources", h.listResources)
	router.GET("/resources/:resource_id", h.getResource)
}

func (h *MCPHandler) listResources(c *gin.Context) {
	orgID := c.GetString("organization_id")
	
	resources, err := h.client.ListResources(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to list resources", "error", err, "org_id", orgID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list resources"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"resources": resources.Resources,
		"total":     resources.Total,
	})
}

func (h *MCPHandler) getResource(c *gin.Context) {
	resourceID := c.Param("resource_id")
	
	// Mock response for now
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"resource": gin.H{
			"resource_id": resourceID,
			"name":        "Mock Resource",
			"type":        "database",
			"status":      "connected",
		},
	})
}
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"github.com/isa-cloud/isa_cloud/internal/config"
	// "github.com/isa-cloud/isa_cloud/internal/gateway/blockchain"  // Temporarily disabled
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// ServiceClients holds all gRPC clients for external services
type ServiceClients struct {
	User       UserClient
	Auth       AuthClient
	Agent      AgentClient
	Model      ModelClient
	MCP        MCPClient
	// Blockchain *blockchain.BlockchainGateway  // Temporarily disabled
	Blockchain interface{}  // Placeholder for blockchain gateway

	// gRPC connections
	userConn  *grpc.ClientConn
	authConn  *grpc.ClientConn
	agentConn *grpc.ClientConn
	modelConn *grpc.ClientConn
	mcpConn   *grpc.ClientConn

	httpClient *http.Client
	config     *config.Config
	logger     *logger.Logger
}

// Client interfaces (simplified for testing)
type UserClient interface {
	GetUser(ctx context.Context, userID string) (*UserResponse, error)
}

type AuthClient interface {
	VerifyToken(ctx context.Context, token string) (*AuthResponse, error)
}

type AgentClient interface {
	ListAgents(ctx context.Context, orgID string) (*AgentsResponse, error)
}

type ModelClient interface {
	ListModels(ctx context.Context, orgID string) (*ModelsResponse, error)
}

type MCPClient interface {
	ListResources(ctx context.Context, orgID string) (*ResourcesResponse, error)
}

// Response types (simplified for testing)
type UserResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	OrgID    string `json:"organization_id"`
}

type AuthResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id"`
	OrgID  string `json:"organization_id"`
}

type AgentsResponse struct {
	Agents []Agent `json:"agents"`
	Total  int     `json:"total"`
}

type Agent struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
}

type ModelsResponse struct {
	Models []Model `json:"models"`
	Total  int     `json:"total"`
}

type Model struct {
	ModelID string `json:"model_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
}

type ResourcesResponse struct {
	Resources []Resource `json:"resources"`
	Total     int        `json:"total"`
}

type Resource struct {
	ResourceID string `json:"resource_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
}

// New creates new service clients
func New(cfg *config.Config, logger *logger.Logger) (*ServiceClients, error) {
	clients := &ServiceClients{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Create real HTTP proxy clients
	clients.User = &httpUserClient{
		baseURL:    fmt.Sprintf("http://%s:%d", cfg.Services.UserService.Host, cfg.Services.UserService.HTTPPort),
		httpClient: clients.httpClient,
		logger:     logger,
	}
	clients.Auth = &httpAuthClient{
		baseURL:    fmt.Sprintf("http://%s:%d", cfg.Services.AuthService.Host, cfg.Services.AuthService.HTTPPort),
		httpClient: clients.httpClient,
		logger:     logger,
	}
	clients.Agent = &httpAgentClient{
		baseURL:    fmt.Sprintf("http://%s:%d", cfg.Services.AgentService.Host, cfg.Services.AgentService.HTTPPort),
		httpClient: clients.httpClient,
		logger:     logger,
	}
	clients.Model = &httpModelClient{
		baseURL:    fmt.Sprintf("http://%s:%d", cfg.Services.ModelService.Host, cfg.Services.ModelService.HTTPPort),
		httpClient: clients.httpClient,
		logger:     logger,
	}
	clients.MCP = &httpMCPClient{
		baseURL:    fmt.Sprintf("http://%s:%d", cfg.Services.MCPService.Host, cfg.Services.MCPService.HTTPPort),
		httpClient: clients.httpClient,
		logger:     logger,
	}

	// Initialize blockchain gateway if enabled
	// TODO: Re-enable blockchain after dependencies are installed
	logger.Info("Blockchain gateway disabled - will be enabled in next phase")

	logger.Info("Service clients initialized (HTTP proxy mode)")
	return clients, nil
}

// CheckConnectivity checks if all services are reachable
func (c *ServiceClients) CheckConnectivity(ctx context.Context) error {
	// For now, always return success in mock mode
	c.logger.Debug("Checking service connectivity (mock mode)")
	
	// Check blockchain gateway connectivity if available
	// if c.Blockchain != nil {
	// 	if err := c.Blockchain.Health(ctx); err != nil {
	// 		c.logger.Warn("Blockchain connectivity check failed", "error", err)
	// 		// Don't fail overall connectivity check for blockchain issues
	// 	}
	// }
	
	return nil
}

// Close closes all gRPC connections
func (c *ServiceClients) Close() error {
	c.logger.Info("Closing service clients")
	
	// Close connections when implemented
	// if c.userConn != nil { c.userConn.Close() }
	// ... other connections

	// Close blockchain gateway if available
	// if c.Blockchain != nil {
	// 	if err := c.Blockchain.Close(); err != nil {
	// 		c.logger.Error("Failed to close blockchain gateway", "error", err)
	// 	}
	// }

	return nil
}

// HTTP proxy implementations

type httpUserClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

func (h *httpUserClient) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	h.logger.Debug("HTTP GetUser called", "user_id", userID, "base_url", h.baseURL)
	
	url := fmt.Sprintf("%s/api/v1/accounts/profile/%s", h.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	var result UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &result, nil
}

type httpAuthClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

func (h *httpAuthClient) VerifyToken(ctx context.Context, token string) (*AuthResponse, error) {
	h.logger.Debug("HTTP VerifyToken called", "token", token[:min(10, len(token))]+"...", "base_url", h.baseURL)
	
	url := fmt.Sprintf("%s/api/v1/auth/verify-token", h.baseURL)
	payload := map[string]string{"token": token}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &result, nil
}

type httpAgentClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

func (h *httpAgentClient) ListAgents(ctx context.Context, orgID string) (*AgentsResponse, error) {
	h.logger.Debug("HTTP ListAgents called", "org_id", orgID, "base_url", h.baseURL)
	
	url := fmt.Sprintf("%s/agents?org_id=%s", h.baseURL, orgID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	var result AgentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &result, nil
}

type httpModelClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

func (h *httpModelClient) ListModels(ctx context.Context, orgID string) (*ModelsResponse, error) {
	h.logger.Debug("HTTP ListModels called", "org_id", orgID, "base_url", h.baseURL)
	
	url := fmt.Sprintf("%s/models?org_id=%s", h.baseURL, orgID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	var result ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &result, nil
}

type httpMCPClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

func (h *httpMCPClient) ListResources(ctx context.Context, orgID string) (*ResourcesResponse, error) {
	h.logger.Debug("HTTP ListResources called", "org_id", orgID, "base_url", h.baseURL)
	
	url := fmt.Sprintf("%s/resources?org_id=%s", h.baseURL, orgID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	
	var result ResourcesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &result, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
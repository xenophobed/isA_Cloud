package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/isa-cloud/isa_cloud/internal/config"
	"github.com/isa-cloud/isa_cloud/internal/gateway/registry"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// DynamicProxy handles dynamic routing to backend services
type DynamicProxy struct {
	config   *config.Config
	logger   *logger.Logger
	proxies  map[string]*httputil.ReverseProxy
	services map[string]*config.ServiceEndpoint
	registry *registry.ConsulRegistry
}

// NewDynamicProxy creates a new dynamic proxy
func NewDynamicProxy(cfg *config.Config, logger *logger.Logger, consulRegistry *registry.ConsulRegistry) *DynamicProxy {
	dp := &DynamicProxy{
		config:   cfg,
		logger:   logger,
		proxies:  make(map[string]*httputil.ReverseProxy),
		services: make(map[string]*config.ServiceEndpoint),
		registry: consulRegistry,
	}

	// Initialize service mappings
	dp.services["users"] = &cfg.Services.UserService
	dp.services["accounts"] = &cfg.Services.UserService  // accounts也路由到user service
	dp.services["auth"] = &cfg.Services.AuthService
	dp.services["agents"] = &cfg.Services.AgentService
	dp.services["models"] = &cfg.Services.ModelService
	dp.services["mcp"] = &cfg.Services.MCPService

	// Create reverse proxies for each service
	for name, svc := range dp.services {
		targetURL := fmt.Sprintf("http://%s:%d", svc.Host, svc.HTTPPort)
		target, err := url.Parse(targetURL)
		if err != nil {
			logger.Error("Failed to parse service URL", "service", name, "url", targetURL, "error", err)
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		
		// Customize the proxy to add logging and error handling
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			logger.Debug("Proxying request", 
				"service", name,
				"method", req.Method,
				"path", req.URL.Path,
				"target", targetURL,
			)
		}

		// Remove CORS headers from downstream services to avoid duplicates
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Remove CORS headers that will be set by the gateway
			resp.Header.Del("Access-Control-Allow-Origin")
			resp.Header.Del("Access-Control-Allow-Methods")
			resp.Header.Del("Access-Control-Allow-Headers")
			resp.Header.Del("Access-Control-Allow-Credentials")
			resp.Header.Del("Access-Control-Max-Age")
			resp.Header.Del("Access-Control-Expose-Headers")
			return nil
		}

		// Add error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("Proxy error", "service", name, "error", err, "path", r.URL.Path)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(fmt.Sprintf(`{"error": "Service unavailable: %s"}`, err.Error())))
		}

		dp.proxies[name] = proxy
	}

	return dp
}

// Handler returns a Gin handler for dynamic proxying
func (dp *DynamicProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only handle /api/v1/* paths
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
			// Not an API path, let other handlers deal with it
			c.Next()
			return
		}

		// Skip gateway management routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/gateway/") {
			c.Next()
			return
		}

		// Extract service name from path
		// Expected format: /api/v1/{service}/...
		parts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, "/api/v1/"), "/")
		if len(parts) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid path"})
			return
		}

		serviceName := parts[0]
		
		// Handle special routing cases for cross-service paths
		if serviceName == "users" && len(parts) >= 3 && parts[2] == "sessions" {
			// Route /api/v1/users/{user_id}/sessions to sessions service
			serviceName = "sessions"
		}
		
		// Try to discover service from Consul first
		if dp.registry != nil {
			instance, err := dp.registry.GetHealthyInstance(serviceName)
			if err == nil {
				targetURL := fmt.Sprintf("http://%s:%d", instance.Host, instance.Port)
				
				// Check if service has SSE tag (for MCP and other streaming services)
				hasSSE := false
				for _, tag := range instance.Tags {
					if tag == "sse" || tag == "streaming" {
						hasSSE = true
						break
					}
				}
				
				dp.logger.Info("Routing to discovered service",
					"service", serviceName,
					"instance", instance.ID,
					"target", targetURL,
					"path", c.Request.URL.Path,
					"sse_enabled", hasSSE,
				)
				
				if hasSSE {
					// Use SSE proxy for services with SSE support
					sseProxy := NewSSEProxy(targetURL, dp.logger)
					sseProxy.Handler()(c)
				} else {
					// Use regular reverse proxy
					target, _ := url.Parse(targetURL)
					proxy := httputil.NewSingleHostReverseProxy(target)
					
					// Remove CORS headers from downstream services
					proxy.ModifyResponse = func(resp *http.Response) error {
						resp.Header.Del("Access-Control-Allow-Origin")
						resp.Header.Del("Access-Control-Allow-Methods")
						resp.Header.Del("Access-Control-Allow-Headers")
						resp.Header.Del("Access-Control-Allow-Credentials")
						resp.Header.Del("Access-Control-Max-Age")
						resp.Header.Del("Access-Control-Expose-Headers")
						return nil
					}
					
					proxy.ServeHTTP(c.Writer, c.Request)
				}
				return
			}
		}
		
		// Fallback to static configuration
		proxy, exists := dp.proxies[serviceName]
		if !exists {
			dp.logger.Warn("No proxy found for service", "service", serviceName, "path", c.Request.URL.Path)
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
			return
		}

		// Log the proxy action
		dp.logger.Info("Routing request (static config)", 
			"service", serviceName,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)

		// Use the reverse proxy to handle the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthCheck checks if all backend services are healthy
func (dp *DynamicProxy) HealthCheck() map[string]bool {
	health := make(map[string]bool)
	
	for name, svc := range dp.services {
		url := fmt.Sprintf("http://%s:%d/health", svc.Host, svc.HTTPPort)
		resp, err := http.Get(url)
		if err != nil {
			health[name] = false
			continue
		}
		resp.Body.Close()
		health[name] = resp.StatusCode == http.StatusOK
	}
	
	return health
}
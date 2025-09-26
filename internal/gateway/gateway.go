package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/isa-cloud/isa_cloud/internal/config"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
	"github.com/isa-cloud/isa_cloud/internal/gateway/middleware"
	"github.com/isa-cloud/isa_cloud/internal/gateway/clients"
	"github.com/isa-cloud/isa_cloud/internal/gateway/proxy"
	"github.com/isa-cloud/isa_cloud/internal/gateway/registry"
	"github.com/isa-cloud/isa_cloud/internal/gateway/blockchain"
)

// Gateway represents the main gateway service
type Gateway struct {
	config            *config.Config
	logger            *logger.Logger
	clients           *clients.ServiceClients
	dynamicProxy      *proxy.DynamicProxy
	registry          *registry.ConsulRegistry
	blockchainGateway *blockchain.Gateway
}

// New creates a new Gateway instance
func New(cfg *config.Config, logger *logger.Logger) (*Gateway, error) {
	// Initialize service clients
	serviceClients, err := clients.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service clients: %w", err)
	}

	// Initialize Consul registry (optional - will work without it)
	var consulRegistry *registry.ConsulRegistry
	consulAddress := "localhost:8500" // Default Consul address
	consulRegistry, err = registry.NewConsulRegistry(consulAddress, logger)
	if err != nil {
		logger.Warn("Failed to connect to Consul, using static configuration", "error", err)
		// Continue without Consul - will use static config
	} else {
		logger.Info("Connected to Consul service registry", "address", consulAddress)
		
		// Register gateway itself with Consul
		err = consulRegistry.RegisterService("gateway", "localhost", cfg.Server.HTTPPort, []string{"api", "gateway"})
		if err != nil {
			logger.Warn("Failed to register gateway with Consul", "error", err)
		}
	}

	// Initialize dynamic proxy
	dynamicProxy := proxy.NewDynamicProxy(cfg, logger, consulRegistry)

	// Initialize blockchain gateway
	var blockchainGateway *blockchain.Gateway
	
	blockchainGateway, err = blockchain.NewGateway(cfg, logger)
	if err != nil {
		logger.Warn("Failed to initialize blockchain gateway", "error", err)
		// Continue without blockchain - optional feature
	} else {
		logger.Info("Blockchain gateway initialized successfully")
		
		// Register blockchain services with Consul if available
		if consulRegistry != nil {
			consulIntegration := blockchain.NewConsulIntegration(blockchainGateway, consulRegistry, logger)
			if err := consulIntegration.RegisterBlockchainServices("localhost", 8545); err != nil {
				logger.Warn("Failed to register blockchain services with Consul", "error", err)
			}
		}
	}

	return &Gateway{
		config:            cfg,
		logger:            logger,
		clients:           serviceClients,
		dynamicProxy:      dynamicProxy,
		registry:          consulRegistry,
		blockchainGateway: blockchainGateway,
	}, nil
}

// SetupHTTPRoutes sets up HTTP routes and middleware
func (g *Gateway) SetupHTTPRoutes() *gin.Engine {
	// Create router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(g.logger))
	router.Use(middleware.RequestID())
	
	// CORS middleware
	if g.config.Security.CORS.Enabled {
		corsConfig := cors.Config{
			AllowOrigins:     g.config.Security.CORS.AllowOrigins,
			AllowMethods:     g.config.Security.CORS.AllowMethods,
			AllowHeaders:     g.config.Security.CORS.AllowHeaders,
			AllowCredentials: g.config.Security.CORS.AllowCredentials,
			MaxAge:           12 * time.Hour,
		}
		router.Use(cors.New(corsConfig))
	}

	// Rate limiting middleware
	if g.config.Security.RateLimit.Enabled {
		router.Use(middleware.RateLimit(
			g.config.Security.RateLimit.RPS,
			g.config.Security.RateLimit.Burst,
		))
	}

	// Health check
	router.GET("/health", g.healthCheck)
	router.GET("/ready", g.readinessCheck)

	// Gateway management routes (these don't go through the proxy)
	gateway := router.Group("/api/v1/gateway")
	gateway.Use(middleware.UnifiedAuthentication(g.clients.Auth, g.registry, g.logger))
	gateway.GET("/services", g.listServices)
	gateway.GET("/metrics", g.getMetrics)
	gateway.GET("/health", g.servicesHealth)
	
	// Blockchain routes (if blockchain gateway is available)
	if g.blockchainGateway != nil {
		blockchainAPI := router.Group("/api/v1/blockchain")
		blockchainAPI.Use(middleware.UnifiedAuthentication(g.clients.Auth, g.registry, g.logger))
		
		// Blockchain endpoints
		blockchainAPI.GET("/status", g.blockchainStatus)
		blockchainAPI.GET("/balance/:address", g.getBalance)
		blockchainAPI.POST("/transaction", g.sendTransaction)
		blockchainAPI.GET("/transaction/:hash", g.getTransaction)
		blockchainAPI.GET("/block/:number", g.getBlock)
	}
	
	// Set up dynamic proxy for service routes
	// Use NoRoute to handle all unmatched requests for dynamic service discovery
	router.NoRoute(g.dynamicProxy.Handler())

	return router
}

// RegisterGRPCServices registers gRPC services
func (g *Gateway) RegisterGRPCServices(server *grpc.Server) {
	// Register gateway management services
	// Note: We'll implement these as needed
}

// GRPCUnaryInterceptor returns a gRPC unary interceptor
func (g *Gateway) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Log request
		g.logger.Debug("gRPC request", 
			"method", info.FullMethod,
			"start_time", start,
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		if err != nil {
			g.logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			g.logger.Debug("gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return resp, err
	}
}

// GRPCStreamInterceptor returns a gRPC stream interceptor
func (g *Gateway) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		g.logger.Debug("gRPC stream started", 
			"method", info.FullMethod,
			"start_time", start,
		)

		err := handler(srv, stream)

		duration := time.Since(start)
		if err != nil {
			g.logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			g.logger.Debug("gRPC stream completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return err
	}
}

// Shutdown gracefully shuts down the gateway
func (g *Gateway) Shutdown(ctx context.Context) error {
	g.logger.Info("Shutting down gateway...")
	
	// Close service clients
	if err := g.clients.Close(); err != nil {
		g.logger.Error("Failed to close service clients", "error", err)
		return err
	}

	g.logger.Info("Gateway shutdown completed")
	return nil
}

// Health check endpoint
func (g *Gateway) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "isa-cloud-gateway",
		"version":   g.config.App.Version,
		"timestamp": time.Now().UTC(),
	})
}

// Readiness check endpoint
func (g *Gateway) readinessCheck(c *gin.Context) {
	// Check service connectivity
	ready := true
	services := make(map[string]bool)

	// Check each service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.clients.CheckConnectivity(ctx); err != nil {
		ready = false
		g.logger.Error("Service connectivity check failed", "error", err)
	}

	// Get individual service status
	services["user_service"] = g.clients.User != nil
	services["auth_service"] = g.clients.Auth != nil
	services["agent_service"] = g.clients.Agent != nil
	services["model_service"] = g.clients.Model != nil
	services["mcp_service"] = g.clients.MCP != nil
	// services["blockchain_gateway"] = g.clients.Blockchain != nil  // Temporarily disabled

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":     ready,
		"services":  services,
		"timestamp": time.Now().UTC(),
	})
}

// List services endpoint
func (g *Gateway) listServices(c *gin.Context) {
	services := []map[string]interface{}{
		{
			"name":      "user_service",
			"host":      g.config.Services.UserService.Host,
			"http_port": g.config.Services.UserService.HTTPPort,
			"grpc_port": g.config.Services.UserService.GRPCPort,
			"status":    "connected",
		},
		{
			"name":      "auth_service", 
			"host":      g.config.Services.AuthService.Host,
			"http_port": g.config.Services.AuthService.HTTPPort,
			"grpc_port": g.config.Services.AuthService.GRPCPort,
			"status":    "connected",
		},
		{
			"name":      "agent_service",
			"host":      g.config.Services.AgentService.Host,
			"http_port": g.config.Services.AgentService.HTTPPort,
			"grpc_port": g.config.Services.AgentService.GRPCPort,
			"status":    "connected",
		},
		{
			"name":      "model_service",
			"host":      g.config.Services.ModelService.Host,
			"http_port": g.config.Services.ModelService.HTTPPort,
			"grpc_port": g.config.Services.ModelService.GRPCPort,
			"status":    "connected",
		},
		{
			"name":      "mcp_service",
			"host":      g.config.Services.MCPService.Host,
			"http_port": g.config.Services.MCPService.HTTPPort,
			"grpc_port": g.config.Services.MCPService.GRPCPort,
			"status":    "connected",
		},
	}

	// Add blockchain service if enabled (temporarily disabled)
	// if g.clients.Blockchain != nil {
	// 	blockchainService := map[string]interface{}{
	// 		"name":         "blockchain_gateway",
	// 		"rpc_endpoint": g.config.Blockchain.RPCEndpoint,
	// 		"chain_id":     g.config.Blockchain.ChainID,
	// 		"network":      g.config.Blockchain.NetworkName,
	// 		"status":       "connected",
	// 	}
	// 	services = append(services, blockchainService)
	// }

	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"total":    len(services),
	})
}

// Get metrics endpoint
func (g *Gateway) getMetrics(c *gin.Context) {
	// TODO: Implement actual metrics collection
	c.JSON(http.StatusOK, gin.H{
		"gateway": map[string]interface{}{
			"uptime":           time.Since(time.Now()).String(),
			"total_requests":   0,
			"active_requests":  0,
			"error_rate":       0.0,
			"average_latency":  "0ms",
		},
		"services": map[string]interface{}{
			"user_service":  map[string]interface{}{"requests": 0, "errors": 0},
			"auth_service":  map[string]interface{}{"requests": 0, "errors": 0},
			"agent_service": map[string]interface{}{"requests": 0, "errors": 0},
			"model_service": map[string]interface{}{"requests": 0, "errors": 0},
			"mcp_service":   map[string]interface{}{"requests": 0, "errors": 0},
		},
	})
}

// Services health endpoint
func (g *Gateway) servicesHealth(c *gin.Context) {
	health := g.dynamicProxy.HealthCheck()
	
	allHealthy := true
	for _, healthy := range health {
		if !healthy {
			allHealthy = false
			break
		}
	}
	
	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}
	
	c.JSON(status, gin.H{
		"healthy":  allHealthy,
		"services": health,
		"timestamp": time.Now().UTC(),
	})
}
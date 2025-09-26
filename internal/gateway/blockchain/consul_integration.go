package blockchain

import (
	"fmt"
	"context"
	
	"github.com/isa-cloud/isa_cloud/internal/gateway/registry"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// ConsulIntegration handles Consul service registration for blockchain services
type ConsulIntegration struct {
	gateway  *Gateway
	registry *registry.ConsulRegistry
	logger   *logger.Logger
}

// NewConsulIntegration creates a new Consul integration for blockchain gateway
func NewConsulIntegration(gateway *Gateway, registry *registry.ConsulRegistry, logger *logger.Logger) *ConsulIntegration {
	return &ConsulIntegration{
		gateway:  gateway,
		registry: registry,
		logger:   logger,
	}
}

// RegisterBlockchainServices registers all blockchain services with Consul
func (ci *ConsulIntegration) RegisterBlockchainServices(host string, port int) error {
	// Register the main blockchain gateway service
	tags := []string{
		"blockchain",
		"gateway",
		"api",
		fmt.Sprintf("version:%s", "1.0.0"),
	}
	
	// Register main gateway
	err := ci.registry.RegisterService(
		"blockchain-gateway",
		host,
		port,
		tags,
	)
	if err != nil {
		return fmt.Errorf("failed to register blockchain gateway: %w", err)
	}
	
	ci.logger.Info("Blockchain gateway registered with Consul")
	
	// Register each blockchain chain as a separate service
	chains := ci.gateway.ListChains()
	for _, chainType := range chains {
		adapter, err := ci.gateway.GetChain(chainType)
		if err != nil {
			ci.logger.Warn("Failed to get chain adapter", "chain", chainType, "error", err)
			continue
		}
		
		if adapter.IsConnected() {
			chainTags := []string{
				"blockchain",
				string(chainType),
				"chain",
			}
			
			// Add chain-specific tags
			if adapter.SupportsSmartContracts() {
				chainTags = append(chainTags, "smart-contracts")
			}
			
			// Register each chain as a service
			serviceID := fmt.Sprintf("blockchain-%s", chainType)
			err = ci.registry.RegisterService(
				serviceID,
				host,
				port, // Same port, different endpoints
				chainTags,
			)
			
			if err != nil {
				ci.logger.Warn("Failed to register chain service", 
					"chain", chainType, 
					"error", err)
			} else {
				ci.logger.Info("Blockchain chain registered with Consul", 
					"chain", chainType,
					"service_id", serviceID)
			}
		}
	}
	
	// Register specific blockchain features as services
	ci.registerFeatureServices(host, port)
	
	return nil
}

// registerFeatureServices registers specific blockchain features as Consul services
func (ci *ConsulIntegration) registerFeatureServices(host string, port int) {
	features := map[string][]string{
		"blockchain-tokens": {"blockchain", "tokens", "erc20", "fungible"},
		"blockchain-nft":    {"blockchain", "nft", "erc721", "non-fungible"},
		"blockchain-defi":   {"blockchain", "defi", "swap", "liquidity"},
		"blockchain-bridge": {"blockchain", "bridge", "cross-chain"},
	}
	
	for serviceName, tags := range features {
		// Check if feature is available
		if ci.isFeatureAvailable(serviceName) {
			err := ci.registry.RegisterService(
				serviceName,
				host,
				port,
				tags,
			)
			
			if err != nil {
				ci.logger.Warn("Failed to register blockchain feature", 
					"feature", serviceName, 
					"error", err)
			} else {
				ci.logger.Info("Blockchain feature registered with Consul", 
					"feature", serviceName)
			}
		}
	}
}

// isFeatureAvailable checks if a blockchain feature is available
func (ci *ConsulIntegration) isFeatureAvailable(feature string) bool {
	// Check if the feature is available based on connected chains
	// and their capabilities
	
	switch feature {
	case "blockchain-tokens", "blockchain-nft", "blockchain-defi":
		// These require at least one chain with smart contract support
		for _, chainType := range ci.gateway.ListChains() {
			if adapter, err := ci.gateway.GetChain(chainType); err == nil {
				if adapter.IsConnected() && adapter.SupportsSmartContracts() {
					return true
				}
			}
		}
		return false
		
	case "blockchain-bridge":
		// Bridge requires at least 2 connected chains
		connectedChains := 0
		for _, chainType := range ci.gateway.ListChains() {
			if adapter, err := ci.gateway.GetChain(chainType); err == nil {
				if adapter.IsConnected() {
					connectedChains++
				}
			}
		}
		return connectedChains >= 2
		
	default:
		return false
	}
}

// UpdateServiceHealth updates the health status of blockchain services
func (ci *ConsulIntegration) UpdateServiceHealth(ctx context.Context) {
	// This would be called periodically to update service health
	health := ci.gateway.HealthCheck(ctx)
	
	for chainName, chainHealth := range health {
		if chainName == "default_chain" || chainName == "total_chains" {
			continue
		}
		
		// Update health status in Consul
		// This is a simplified version - in production you'd update
		// the actual health check status
		if chainStatus, ok := chainHealth.(map[string]interface{}); ok {
			if connected, ok := chainStatus["connected"].(bool); ok && !connected {
				ci.logger.Warn("Chain disconnected, updating Consul", "chain", chainName)
				// Here you would update the service health check in Consul
			}
		}
	}
}

// DiscoverBlockchainNodes discovers other blockchain nodes from Consul
func (ci *ConsulIntegration) DiscoverBlockchainNodes(chainType string) ([]*registry.ServiceInstance, error) {
	serviceName := fmt.Sprintf("blockchain-%s-node", chainType)
	return ci.registry.DiscoverService(serviceName)
}

// RegisterBlockchainNode registers a blockchain node with Consul
func (ci *ConsulIntegration) RegisterBlockchainNode(chainType string, host string, port int, nodeType string) error {
	tags := []string{
		"blockchain",
		"node",
		string(chainType),
		nodeType, // "validator", "full", "light"
	}
	
	serviceID := fmt.Sprintf("blockchain-%s-node-%s-%d", chainType, host, port)
	
	return ci.registry.RegisterService(
		serviceID,
		host,
		port,
		tags,
	)
}

// DeregisterAll deregisters all blockchain services from Consul
func (ci *ConsulIntegration) DeregisterAll() error {
	services := []string{
		"blockchain-gateway",
		"blockchain-tokens",
		"blockchain-nft",
		"blockchain-defi",
		"blockchain-bridge",
	}
	
	// Also deregister chain-specific services
	for _, chainType := range ci.gateway.ListChains() {
		services = append(services, fmt.Sprintf("blockchain-%s", chainType))
	}
	
	var lastErr error
	for _, service := range services {
		if err := ci.registry.DeregisterService(service); err != nil {
			ci.logger.Warn("Failed to deregister service", 
				"service", service, 
				"error", err)
			lastErr = err
		}
	}
	
	return lastErr
}
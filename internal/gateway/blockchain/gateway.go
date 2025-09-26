package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	
	"github.com/isa-cloud/isa_cloud/internal/config"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// Gateway implements the BlockchainGateway interface
type Gateway struct {
	config        *config.Config
	logger        *logger.Logger
	chains        map[ChainType]BlockchainAdapter
	defaultChain  ChainType
	mu            sync.RWMutex
}

// NewGateway creates a new blockchain gateway
func NewGateway(cfg *config.Config, logger *logger.Logger) (*Gateway, error) {
	g := &Gateway{
		config: cfg,
		logger: logger,
		chains: make(map[ChainType]BlockchainAdapter),
	}
	
	// Initialize based on configuration
	if err := g.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize blockchain gateway: %w", err)
	}
	
	return g, nil
}

// initialize sets up the blockchain connections based on configuration
func (g *Gateway) initialize() error {
	// Check if blockchain is enabled
	if !g.config.Blockchain.Enabled {
		g.logger.Info("Blockchain integration is disabled")
		return nil
	}

	// Get default chain configuration
	defaultChainName := g.config.Blockchain.DefaultChain
	if defaultChainName == "" {
		defaultChainName = "isa_chain"
	}

	chainConfig, exists := g.config.Blockchain.Chains[defaultChainName]
	if !exists {
		return fmt.Errorf("default chain %s not found in configuration", defaultChainName)
	}

	// Initialize isA_Chain as the primary chain
	isaConfig := &ISAChainConfig{
		RPCEndpoint:   chainConfig.RPCEndpoint,
		ChainID:       chainConfig.ChainID,
		NetworkName:   chainConfig.NetworkName,
		PrivateKey:    chainConfig.PrivateKey,
		ConsensusType: "hybrid", // Default for isA_Chain
	}

	// Set consensus type from custom settings if available
	if consensusType, ok := chainConfig.CustomSettings["consensus_type"].(string); ok {
		isaConfig.ConsensusType = consensusType
	}
	
	isaAdapter := NewISAChainAdapter(isaConfig, g.logger)
	
	// Connect to isA_Chain
	ctx := context.Background()
	if err := isaAdapter.Connect(ctx); err != nil {
		g.logger.Warn("Failed to connect to isA_Chain", "error", err)
		// Don't fail initialization, allow lazy connection
	} else {
		g.logger.Info("Connected to isA_Chain successfully")
	}
	
	// Register the adapter
	if err := g.RegisterChain(ChainTypeISA, isaAdapter); err != nil {
		return fmt.Errorf("failed to register isA_Chain: %w", err)
	}
	
	// Set isA_Chain as default
	if err := g.SetDefaultChain(ChainTypeISA); err != nil {
		return fmt.Errorf("failed to set default chain: %w", err)
	}
	
	// TODO: Add other chains based on configuration
	// Example: Ethereum, Solana, Polygon adapters
	
	return nil
}

// getContractAddress retrieves contract address from configuration
func (g *Gateway) getContractAddress(contractType string) (string, error) {
	// Get default chain configuration
	defaultChainName := g.config.Blockchain.DefaultChain
	if defaultChainName == "" {
		defaultChainName = "isa_chain"
	}

	chainConfig, exists := g.config.Blockchain.Chains[defaultChainName]
	if !exists {
		return "", fmt.Errorf("default chain %s not found in configuration", defaultChainName)
	}

	contracts := chainConfig.Contracts
	
	switch contractType {
	case "reward_token", "isa_token":
		return contracts.ISATokenAddress, nil
	case "billing", "usage_billing":
		return contracts.UsageBillingAddress, nil
	case "service_nft", "isa_nft":
		return contracts.ISANFTAddress, nil
	case "service_registry":
		return contracts.ServiceRegistryAddress, nil
	case "dex", "simple_dex":
		return contracts.SimpleDEXAddress, nil
	case "nft_marketplace":
		return contracts.NFTMarketplaceAddress, nil
	default:
		return "", fmt.Errorf("unknown contract type: %s", contractType)
	}
}

// RegisterChain registers a new blockchain adapter
func (g *Gateway) RegisterChain(chainType ChainType, adapter BlockchainAdapter) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if _, exists := g.chains[chainType]; exists {
		return fmt.Errorf("chain %s already registered", chainType)
	}
	
	g.chains[chainType] = adapter
	g.logger.Info("Registered blockchain adapter", "chain", chainType)
	
	return nil
}

// GetChain returns a specific blockchain adapter
func (g *Gateway) GetChain(chainType ChainType) (BlockchainAdapter, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	adapter, exists := g.chains[chainType]
	if !exists {
		return nil, fmt.Errorf("chain %s not found", chainType)
	}
	
	return adapter, nil
}

// ListChains returns all registered chain types
func (g *Gateway) ListChains() []ChainType {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	chains := make([]ChainType, 0, len(g.chains))
	for chainType := range g.chains {
		chains = append(chains, chainType)
	}
	
	return chains
}

// SetDefaultChain sets the default chain for operations
func (g *Gateway) SetDefaultChain(chainType ChainType) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if _, exists := g.chains[chainType]; !exists {
		return fmt.Errorf("chain %s not registered", chainType)
	}
	
	g.defaultChain = chainType
	g.logger.Info("Set default chain", "chain", chainType)
	
	return nil
}

// GetDefaultChain returns the default blockchain adapter
func (g *Gateway) GetDefaultChain() (BlockchainAdapter, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if g.defaultChain == "" {
		return nil, fmt.Errorf("no default chain set")
	}
	
	return g.chains[g.defaultChain], nil
}

// BridgeTokens bridges tokens between chains
func (g *Gateway) BridgeTokens(ctx context.Context, fromChain, toChain ChainType, token string, amount *big.Int) (string, error) {
	// Get source and destination chains
	sourceChain, err := g.GetChain(fromChain)
	if err != nil {
		return "", fmt.Errorf("source chain not found: %w", err)
	}
	
	destChain, err := g.GetChain(toChain)
	if err != nil {
		return "", fmt.Errorf("destination chain not found: %w", err)
	}
	
	g.logger.Info("Bridging tokens",
		"from", fromChain,
		"to", toChain,
		"token", token,
		"amount", amount)
	
	// Implementation would involve:
	// 1. Lock tokens on source chain
	// 2. Wait for confirmation
	// 3. Mint/unlock tokens on destination chain
	// 4. Return bridge transaction ID
	
	// This is a simplified mock implementation
	_ = sourceChain
	_ = destChain
	
	return fmt.Sprintf("bridge_%d", amount.Int64()), nil
}

// GetCrossChainBalance gets balance across all registered chains
func (g *Gateway) GetCrossChainBalance(ctx context.Context, address string) (map[ChainType]*big.Int, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	balances := make(map[ChainType]*big.Int)
	
	for chainType, adapter := range g.chains {
		if !adapter.IsConnected() {
			g.logger.Debug("Chain not connected, skipping", "chain", chainType)
			continue
		}
		
		balance, err := adapter.GetBalance(ctx, address)
		if err != nil {
			g.logger.Warn("Failed to get balance", 
				"chain", chainType, 
				"error", err)
			balances[chainType] = big.NewInt(0)
		} else {
			balances[chainType] = balance
		}
	}
	
	return balances, nil
}

// High-level operations using the default chain

// GetTokenBalance gets token balance on default chain
func (g *Gateway) GetTokenBalance(ctx context.Context, tokenAddress, accountAddress string) (*big.Int, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return nil, err
	}
	
	return chain.GetTokenBalance(ctx, tokenAddress, accountAddress)
}

// MintRewardTokens mints reward tokens on default chain
func (g *Gateway) MintRewardTokens(ctx context.Context, userAddress string, amount *big.Int, reason string) (string, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return "", err
	}
	
	g.logger.Info("Minting reward tokens",
		"user", userAddress,
		"amount", amount,
		"reason", reason)
	
	// Get reward token contract address from config
	rewardTokenAddress, err := g.getContractAddress("reward_token")
	if err != nil {
		return "", fmt.Errorf("failed to get reward token contract address: %w", err)
	}
	if rewardTokenAddress == "" {
		return "", fmt.Errorf("reward token contract not configured")
	}
	
	// Mint tokens (requires minter role)
	return chain.ExecuteContract(ctx, rewardTokenAddress, "mint", userAddress, amount)
}

// DeductServiceTokens deducts tokens for service usage
func (g *Gateway) DeductServiceTokens(ctx context.Context, userAddress string, amount *big.Int, serviceID string) (string, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return "", err
	}
	
	g.logger.Info("Deducting service tokens",
		"user", userAddress,
		"amount", amount,
		"service", serviceID)
	
	// Get billing contract address
	billingAddress, err := g.getContractAddress("billing")
	if err != nil {
		return "", fmt.Errorf("failed to get billing contract address: %w", err)
	}
	if billingAddress == "" {
		return "", fmt.Errorf("billing contract not configured")
	}
	
	// Deduct tokens for service
	return chain.ExecuteContract(ctx, billingAddress, "chargeUser", userAddress, amount, serviceID)
}

// MintServiceCertificate mints an NFT certificate for service completion
func (g *Gateway) MintServiceCertificate(ctx context.Context, userAddress string, serviceID string, metadata map[string]string) (string, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return "", err
	}
	
	g.logger.Info("Minting service certificate NFT",
		"user", userAddress,
		"service", serviceID)
	
	// Get NFT contract address
	nftAddress, err := g.getContractAddress("service_nft")
	if err != nil {
		return "", fmt.Errorf("failed to get service NFT contract address: %w", err)
	}
	if nftAddress == "" {
		return "", fmt.Errorf("service NFT contract not configured")
	}
	
	// Create token URI with metadata
	tokenURI := fmt.Sprintf("ipfs://service_%s_%s", serviceID, userAddress)
	
	// Mint NFT
	return chain.MintNFT(ctx, nftAddress, userAddress, tokenURI)
}

// VerifyServiceAccess verifies if user has access to a service
func (g *Gateway) VerifyServiceAccess(ctx context.Context, userAddress string, serviceID string) (bool, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return false, err
	}
	
	// Get service registry contract
	registryAddress, err := g.getContractAddress("service_registry")
	if err != nil {
		return false, fmt.Errorf("failed to get service registry contract address: %w", err)
	}
	if registryAddress == "" {
		return false, fmt.Errorf("service registry contract not configured")
	}
	
	// Check access
	result, err := chain.CallContract(ctx, registryAddress, "hasAccess", userAddress, serviceID)
	if err != nil {
		return false, err
	}
	
	if len(result) > 0 {
		if hasAccess, ok := result[0].(bool); ok {
			return hasAccess, nil
		}
	}
	
	return false, nil
}

// SwapTokensForService swaps tokens to pay for service
func (g *Gateway) SwapTokensForService(ctx context.Context, userAddress string, tokenIn string, serviceTokenAmount *big.Int) (string, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return "", err
	}
	
	g.logger.Info("Swapping tokens for service payment",
		"user", userAddress,
		"tokenIn", tokenIn,
		"amount", serviceTokenAmount)
	
	// Get DEX contract address
	dexAddress, err := g.getContractAddress("dex")
	if err != nil {
		return "", fmt.Errorf("failed to get DEX contract address: %w", err)
	}
	if dexAddress == "" {
		return "", fmt.Errorf("DEX contract not configured")
	}
	
	// Get service token address
	serviceToken, err := g.getContractAddress("isa_token")
	if err != nil {
		return "", fmt.Errorf("failed to get service token contract address: %w", err)
	}
	
	// Perform swap
	return chain.SwapTokens(ctx, dexAddress, tokenIn, serviceToken, serviceTokenAmount)
}

// GetServicePricing gets the current pricing for a service
func (g *Gateway) GetServicePricing(ctx context.Context, serviceID string) (*big.Int, error) {
	chain, err := g.GetDefaultChain()
	if err != nil {
		return nil, err
	}
	
	// Get pricing contract
	pricingAddress, err := g.getContractAddress("service_registry") // Use service registry for pricing
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing contract address: %w", err)
	}
	if pricingAddress == "" {
		return nil, fmt.Errorf("pricing contract not configured")
	}
	
	// Get price
	result, err := chain.CallContract(ctx, pricingAddress, "getServicePrice", serviceID)
	if err != nil {
		return nil, err
	}
	
	if len(result) > 0 {
		if price, ok := result[0].(*big.Int); ok {
			return price, nil
		}
	}
	
	return nil, fmt.Errorf("failed to get service pricing")
}

// HealthCheck performs health check on all registered chains
func (g *Gateway) HealthCheck(ctx context.Context) map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	health := make(map[string]interface{})
	
	for chainType, adapter := range g.chains {
		chainHealth := map[string]interface{}{
			"connected": adapter.IsConnected(),
			"type":      string(chainType),
		}
		
		if adapter.IsConnected() {
			// Try to get block number as health indicator
			if blockNum, err := adapter.GetBlockNumber(ctx); err == nil {
				chainHealth["block_number"] = blockNum
			} else {
				chainHealth["error"] = err.Error()
			}
		}
		
		health[string(chainType)] = chainHealth
	}
	
	health["default_chain"] = string(g.defaultChain)
	health["total_chains"] = len(g.chains)
	
	return health
}

// Close closes all blockchain connections
func (g *Gateway) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	var lastErr error
	for chainType, adapter := range g.chains {
		if err := adapter.Disconnect(); err != nil {
			g.logger.Error("Failed to disconnect from chain", 
				"chain", chainType, 
				"error", err)
			lastErr = err
		}
	}
	
	return lastErr
}
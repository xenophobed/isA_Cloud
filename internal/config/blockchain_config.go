package config

import "fmt"

// BlockchainConfig holds blockchain-related configuration (chain-agnostic)
type BlockchainConfig struct {
	// General settings
	Enabled     bool   `mapstructure:"enabled" json:"enabled"`
	DefaultChain string `mapstructure:"default_chain" json:"default_chain"` // "isa_chain", "ethereum", "solana", etc.
	
	// Chain configurations
	Chains map[string]ChainConfig `mapstructure:"chains" json:"chains"`
	
	// Cross-chain settings
	BridgeEnabled bool `mapstructure:"bridge_enabled" json:"bridge_enabled"`
	
	// Service settings
	HealthCheckInterval string `mapstructure:"health_check_interval" json:"health_check_interval"`
}

// ChainConfig represents configuration for a specific blockchain
type ChainConfig struct {
	Type        string `mapstructure:"type" json:"type"`               // "isa_chain", "ethereum", "solana", etc.
	RPCEndpoint string `mapstructure:"rpc_endpoint" json:"rpc_endpoint"`
	WSEndpoint  string `mapstructure:"ws_endpoint" json:"ws_endpoint"`
	ChainID     int64  `mapstructure:"chain_id" json:"chain_id"`
	NetworkName string `mapstructure:"network_name" json:"network_name"`
	
	// Authentication
	PrivateKey string `mapstructure:"private_key" json:"-"` // Don't log private key
	PublicKey  string `mapstructure:"public_key" json:"public_key"`
	
	// Contract addresses (if applicable)
	Contracts ContractAddresses `mapstructure:"contracts" json:"contracts"`
	
	// Transaction settings
	GasLimit      uint64 `mapstructure:"gas_limit" json:"gas_limit"`
	GasPrice      string `mapstructure:"gas_price" json:"gas_price"`
	Confirmations int    `mapstructure:"confirmations" json:"confirmations"`
	
	// Chain-specific settings
	CustomSettings map[string]interface{} `mapstructure:"custom" json:"custom"`
}

// ContractAddresses holds deployed contract addresses
type ContractAddresses struct {
	ISATokenAddress         string `mapstructure:"isa_token" json:"isa_token"`
	ISANFTAddress          string `mapstructure:"isa_nft" json:"isa_nft"`
	NFTMarketplaceAddress  string `mapstructure:"nft_marketplace" json:"nft_marketplace"`
	SimpleDEXAddress       string `mapstructure:"simple_dex" json:"simple_dex"`
	ServiceRegistryAddress string `mapstructure:"service_registry" json:"service_registry"`
	UsageBillingAddress    string `mapstructure:"usage_billing" json:"usage_billing"`
}

// DefaultBlockchainConfig returns default blockchain configuration
func DefaultBlockchainConfig() *BlockchainConfig {
	return &BlockchainConfig{
		Enabled:      true,
		DefaultChain: "isa_chain",
		Chains: map[string]ChainConfig{
			"isa_chain": {
				Type:        "isa_chain",
				RPCEndpoint: "http://localhost:8545",
				ChainID:     1337,
				NetworkName: "isA_Chain Local",
				Contracts: ContractAddresses{
					// These will be filled after contract deployment
					ISATokenAddress:         "",
					ISANFTAddress:          "",
					NFTMarketplaceAddress:  "",
					SimpleDEXAddress:       "",
					ServiceRegistryAddress: "",
					UsageBillingAddress:    "",
				},
				GasLimit:      300000,
				GasPrice:      "20000000000", // ISA chain gas pricing
				Confirmations: 1,
				CustomSettings: map[string]interface{}{
					"consensus_type": "hybrid",
					"shard_id":      0,
				},
			},
		},
		BridgeEnabled:       false,
		HealthCheckInterval: "30s",
	}
}

// Validate validates the blockchain configuration
func (bc *BlockchainConfig) Validate() error {
	if !bc.Enabled {
		return nil // Skip validation if disabled
	}
	
	if bc.DefaultChain == "" {
		return fmt.Errorf("default_chain is required")
	}
	
	if len(bc.Chains) == 0 {
		return fmt.Errorf("at least one chain must be configured")
	}
	
	// Validate default chain exists
	if _, exists := bc.Chains[bc.DefaultChain]; !exists {
		return fmt.Errorf("default chain %s not found in configured chains", bc.DefaultChain)
	}
	
	// Validate each chain configuration
	for name, chain := range bc.Chains {
		if chain.RPCEndpoint == "" {
			return fmt.Errorf("chain %s: rpc_endpoint is required", name)
		}
		
		if chain.ChainID <= 0 {
			return fmt.Errorf("chain %s: chain_id must be positive", name)
		}
		
		if chain.PrivateKey == "" {
			return fmt.Errorf("chain %s: private_key is required", name)
		}
		
		if chain.GasLimit == 0 {
			return fmt.Errorf("chain %s: gas_limit must be positive", name)
		}
	}
	
	return nil
}
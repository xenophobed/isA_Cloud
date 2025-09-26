package blockchain

import (
	"context"
	"math/big"
	"time"
)

// ChainType represents different blockchain types
type ChainType string

const (
	ChainTypeISA      ChainType = "isa_chain"
	ChainTypeEthereum ChainType = "ethereum"
	ChainTypeSolana   ChainType = "solana"
	ChainTypePolygon  ChainType = "polygon"
	ChainTypeBSC      ChainType = "bsc"
	ChainTypeCustom   ChainType = "custom"
)

// Transaction represents a generic blockchain transaction
type Transaction struct {
	Hash        string
	From        string
	To          string
	Value       *big.Int
	Data        []byte
	GasLimit    uint64
	GasPrice    *big.Int
	Nonce       uint64
	ChainID     *big.Int
	BlockNumber uint64
	Status      TransactionStatus
	Timestamp   time.Time
}

// TransactionStatus represents transaction status
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusConfirmed TransactionStatus = "confirmed"
	StatusFailed    TransactionStatus = "failed"
)

// TokenInfo represents token information
type TokenInfo struct {
	Address     string
	Symbol      string
	Name        string
	Decimals    uint8
	TotalSupply *big.Int
	ChainType   ChainType
}

// NFTMetadata represents NFT metadata
type NFTMetadata struct {
	TokenID     string
	Owner       string
	TokenURI    string
	Name        string
	Description string
	Image       string
	Attributes  map[string]interface{}
}

// DeFiPool represents a DeFi liquidity pool
type DeFiPool struct {
	Address    string
	Token0     string
	Token1     string
	Reserve0   *big.Int
	Reserve1   *big.Int
	TotalSupply *big.Int
	APY        float64
}

// BlockchainClient is the main interface for blockchain operations
type BlockchainClient interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetChainType() ChainType
	GetChainID() (*big.Int, error)
	GetBlockNumber(ctx context.Context) (uint64, error)
	
	// Account operations
	GetBalance(ctx context.Context, address string) (*big.Int, error)
	GetTokenBalance(ctx context.Context, tokenAddress, accountAddress string) (*big.Int, error)
	GetNonce(ctx context.Context, address string) (uint64, error)
	
	// Transaction operations
	SendTransaction(ctx context.Context, tx *Transaction) (string, error)
	GetTransaction(ctx context.Context, txHash string) (*Transaction, error)
	WaitForConfirmation(ctx context.Context, txHash string, confirmations int) (*Transaction, error)
	EstimateGas(ctx context.Context, tx *Transaction) (uint64, error)
	
	// Smart contract operations
	CallContract(ctx context.Context, contractAddress string, method string, args ...interface{}) ([]interface{}, error)
	ExecuteContract(ctx context.Context, contractAddress string, method string, args ...interface{}) (string, error)
	DeployContract(ctx context.Context, bytecode []byte, args ...interface{}) (string, error)
	
	// Token operations
	GetTokenInfo(ctx context.Context, tokenAddress string) (*TokenInfo, error)
	TransferToken(ctx context.Context, tokenAddress, to string, amount *big.Int) (string, error)
	ApproveToken(ctx context.Context, tokenAddress, spender string, amount *big.Int) (string, error)
	
	// NFT operations
	MintNFT(ctx context.Context, contractAddress, to, tokenURI string) (string, error)
	TransferNFT(ctx context.Context, contractAddress, from, to, tokenID string) (string, error)
	GetNFTMetadata(ctx context.Context, contractAddress, tokenID string) (*NFTMetadata, error)
	GetNFTOwner(ctx context.Context, contractAddress, tokenID string) (string, error)
	
	// DeFi operations
	GetPoolInfo(ctx context.Context, poolAddress string) (*DeFiPool, error)
	SwapTokens(ctx context.Context, poolAddress string, tokenIn, tokenOut string, amountIn *big.Int) (string, error)
	AddLiquidity(ctx context.Context, poolAddress string, token0Amount, token1Amount *big.Int) (string, error)
	RemoveLiquidity(ctx context.Context, poolAddress string, lpAmount *big.Int) (string, error)
}

// BlockchainAdapter is the interface for chain-specific adapters
type BlockchainAdapter interface {
	BlockchainClient
	
	// Chain-specific configuration
	Configure(config map[string]interface{}) error
	ValidateConfig() error
	
	// Chain-specific features
	GetNativeTokenSymbol() string
	GetExplorerURL(txHash string) string
	SupportsSmartContracts() bool
	SupportsTokenStandard(standard string) bool
}

// BlockchainEventListener represents blockchain event listener
type BlockchainEventListener interface {
	// Event subscription
	SubscribeToTransfers(ctx context.Context, address string, handler func(*Transaction)) error
	SubscribeToContract(ctx context.Context, contractAddress string, eventName string, handler func(map[string]interface{})) error
	SubscribeToBlocks(ctx context.Context, handler func(uint64)) error
	
	// Unsubscribe
	UnsubscribeAll() error
}

// BlockchainGateway manages multiple blockchain connections
type BlockchainGateway interface {
	// Chain management
	RegisterChain(chainType ChainType, adapter BlockchainAdapter) error
	GetChain(chainType ChainType) (BlockchainAdapter, error)
	ListChains() []ChainType
	SetDefaultChain(chainType ChainType) error
	GetDefaultChain() (BlockchainAdapter, error)
	
	// Cross-chain operations
	BridgeTokens(ctx context.Context, fromChain, toChain ChainType, token string, amount *big.Int) (string, error)
	GetCrossChainBalance(ctx context.Context, address string) (map[ChainType]*big.Int, error)
}

// ServiceRegistry represents on-chain service registry
type ServiceRegistry interface {
	RegisterService(ctx context.Context, serviceID, endpoint string, metadata map[string]string) (string, error)
	GetService(ctx context.Context, serviceID string) (map[string]interface{}, error)
	UpdateService(ctx context.Context, serviceID string, updates map[string]string) (string, error)
	UnregisterService(ctx context.Context, serviceID string) (string, error)
	ListServices(ctx context.Context, filter map[string]string) ([]map[string]interface{}, error)
}

// UsageBilling represents on-chain usage billing
type UsageBilling interface {
	RecordUsage(ctx context.Context, userID, serviceID string, units uint64) (string, error)
	GetUsage(ctx context.Context, userID, serviceID string, period time.Duration) (uint64, error)
	BillUser(ctx context.Context, userID string, amount *big.Int) (string, error)
	GetBillingHistory(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error)
}
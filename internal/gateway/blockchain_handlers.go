package gateway

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isa-cloud/isa_cloud/internal/gateway/blockchain"
)

// blockchainStatus returns the status of blockchain connections
func (g *Gateway) blockchainStatus(c *gin.Context) {
	if g.blockchainGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blockchain gateway not available",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chain, err := g.blockchainGateway.GetChain(blockchain.ChainTypeISA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blockchain adapter",
		})
		return
	}

	blockNumber, err := chain.GetBlockNumber(ctx)
	if err != nil {
		g.logger.Error("Failed to get block number", "error", err)
	}

	chainID, err := chain.GetChainID()
	if err != nil {
		g.logger.Error("Failed to get chain ID", "error", err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"chain_type":    string(blockchain.ChainTypeISA),
		"connected":     chain.IsConnected(),
		"block_number":  blockNumber,
		"chain_id":      chainID.String(),
		"timestamp":     time.Now().Unix(),
	})
}

// getBalance returns the balance for a given address
func (g *Gateway) getBalance(c *gin.Context) {
	if g.blockchainGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blockchain gateway not available",
		})
		return
	}

	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Address parameter is required",
		})
		return
	}

	chain, err := g.blockchainGateway.GetChain(blockchain.ChainTypeISA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blockchain adapter",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	balance, err := chain.GetBalance(ctx, address)
	if err != nil {
		g.logger.Error("Failed to get balance", "error", err, "address", address)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get balance",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"balance": balance.String(),
		"wei":     balance.String(),
		"eth":     new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18)).String(),
	})
}

// sendTransaction sends a transaction to the blockchain
func (g *Gateway) sendTransaction(c *gin.Context) {
	if g.blockchainGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blockchain gateway not available",
		})
		return
	}

	var req struct {
		To       string `json:"to" binding:"required"`
		Value    string `json:"value"`
		Data     string `json:"data"`
		GasLimit uint64 `json:"gasLimit"`
		GasPrice string `json:"gasPrice"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Parse value
	value := big.NewInt(0)
	if req.Value != "" {
		var ok bool
		value, ok = new(big.Int).SetString(req.Value, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid value format",
			})
			return
		}
	}

	// Parse gas price
	gasPrice := big.NewInt(20000000000) // 20 gwei default
	if req.GasPrice != "" {
		var ok bool
		gasPrice, ok = new(big.Int).SetString(req.GasPrice, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid gas price format",
			})
			return
		}
	}

	// Create transaction
	tx := &blockchain.Transaction{
		To:       req.To,
		Value:    value,
		Data:     []byte(req.Data),
		GasLimit: req.GasLimit,
		GasPrice: gasPrice,
	}

	if tx.GasLimit == 0 {
		tx.GasLimit = 21000 // Default gas limit
	}

	chain, err := g.blockchainGateway.GetChain(blockchain.ChainTypeISA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blockchain adapter",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	txHash, err := chain.SendTransaction(ctx, tx)
	if err != nil {
		g.logger.Error("Failed to send transaction", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send transaction",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_hash": txHash,
		"status":          "pending",
	})
}

// getTransaction returns transaction details by hash
func (g *Gateway) getTransaction(c *gin.Context) {
	if g.blockchainGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blockchain gateway not available",
		})
		return
	}

	txHash := c.Param("hash")
	if txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Transaction hash parameter is required",
		})
		return
	}

	chain, err := g.blockchainGateway.GetChain(blockchain.ChainTypeISA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blockchain adapter",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := chain.GetTransaction(ctx, txHash)
	if err != nil {
		g.logger.Error("Failed to get transaction", "error", err, "hash", txHash)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get transaction",
		})
		return
	}

	if tx == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Transaction not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hash":         tx.Hash,
		"from":         tx.From,
		"to":           tx.To,
		"value":        tx.Value.String(),
		"gas_limit":    tx.GasLimit,
		"gas_price":    tx.GasPrice.String(),
		"nonce":        tx.Nonce,
		"block_number": tx.BlockNumber,
		"status":       string(tx.Status),
		"timestamp":    tx.Timestamp.Unix(),
	})
}

// getBlock returns block details by number
func (g *Gateway) getBlock(c *gin.Context) {
	if g.blockchainGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blockchain gateway not available",
		})
		return
	}

	blockNumberStr := c.Param("number")
	if blockNumberStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Block number parameter is required",
		})
		return
	}

	var blockNumber uint64
	if blockNumberStr == "latest" {
		blockNumber = 0 // 0 indicates latest block
	} else {
		var err error
		blockNumber, err = strconv.ParseUint(blockNumberStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid block number format",
			})
			return
		}
	}

	chain, err := g.blockchainGateway.GetChain(blockchain.ChainTypeISA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blockchain adapter",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	currentBlockNumber, err := chain.GetBlockNumber(ctx)
	if err != nil {
		g.logger.Error("Failed to get current block number", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get block information",
		})
		return
	}

	// If latest block requested or block number is 0, use current block number
	if blockNumber == 0 {
		blockNumber = currentBlockNumber
	}

	// For now, return basic block information since GetBlock is not in the interface
	c.JSON(http.StatusOK, gin.H{
		"number":      blockNumber,
		"current":     currentBlockNumber,
		"timestamp":   time.Now().Unix(),
		"status":      "available",
	})
}
# Blockchain Integration Guide

## 架构概述

当前的区块链集成采用 **Gateway中心化模式**：

```
微服务 → API Gateway → Blockchain Gateway → isA_Chain
```

### 设计优势

1. **简化微服务开发** - 微服务无需直接处理区块链复杂性
2. **统一访问控制** - Gateway统一管理区块链访问权限
3. **易于升级维护** - 区块链逻辑集中在Gateway中
4. **降低配置复杂度** - 微服务无需配置区块链连接

## 微服务集成指南

### 1. 安装区块链客户端

每个微服务通过 `BlockchainClient` 访问区块链功能：

```python
from core.blockchain_client import BlockchainClient

# 初始化客户端
blockchain_client = BlockchainClient(
    gateway_url="http://localhost:8000",
    auth_token=your_auth_token
)
```

### 2. 可用的区块链操作

#### 查询操作
```python
# 获取区块链状态
status = await blockchain_client.get_status()

# 查询账户余额
balance = await blockchain_client.get_balance(address)

# 获取交易详情
tx = await blockchain_client.get_transaction(tx_hash)

# 获取区块信息
block = await blockchain_client.get_block("latest")
```

#### 交易操作
```python
# 发送交易
result = await blockchain_client.send_transaction(
    to=recipient_address,
    value="1000000000000000000",  # 1 token in wei
    data="optional_data"
)

# 为服务收费
tx = await blockchain_client.charge_for_service(
    user_address=user_address,
    amount=amount,
    service_id=service_id
)

# 发送奖励
tx = await blockchain_client.reward_user(
    user_address=user_address,
    amount=amount,
    reason="completion_bonus"
)
```

#### 验证操作
```python
# 验证支付
is_valid = await blockchain_client.verify_payment(
    tx_hash=tx_hash,
    expected_amount=amount
)

# 检查服务访问权限
has_access = await blockchain_client.check_service_access(
    user_address=user_address,
    service_id=service_id
)
```

## 微服务示例

### Payment Service 集成

Payment Service 已集成区块链功能，提供以下端点：

```bash
# 创建区块链支付
POST /api/v1/payments/blockchain/payment
{
    "user_address": "0x...",
    "amount": "1000000000000000000",
    "order_id": "ORDER123",
    "service_id": "SERVICE456"
}

# 验证支付
GET /api/v1/payments/blockchain/payment/{tx_hash}/verify?amount=1000000000000000000

# 发起退款
POST /api/v1/payments/blockchain/refund
{
    "user_address": "0x...",
    "amount": "500000000000000000",
    "order_id": "ORDER123",
    "reason": "Customer request"
}

# 检查订阅状态
GET /api/v1/payments/blockchain/subscription/{user_address}/{service_id}
```

### 其他微服务集成模板

```python
# account_service/blockchain_features.py
from core.blockchain_client import BlockchainClient

async def verify_user_wallet(user_id: str, wallet_address: str):
    """验证用户钱包地址"""
    client = BlockchainClient()
    
    # 验证地址有效性
    balance = await client.get_balance(wallet_address)
    if balance:
        # 保存到用户账户
        return {"verified": True, "address": wallet_address}
    return {"verified": False}

async def get_user_token_balance(wallet_address: str):
    """获取用户代币余额"""
    client = BlockchainClient()
    balance_info = await client.get_balance(wallet_address)
    return {
        "address": wallet_address,
        "balance_wei": balance_info["balance"],
        "balance_token": balance_info["eth"]  # 转换后的值
    }
```

## API Gateway 区块链端点

Gateway 提供以下区块链 API：

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/blockchain/status` | GET | 获取区块链连接状态 |
| `/api/v1/blockchain/balance/:address` | GET | 查询地址余额 |
| `/api/v1/blockchain/transaction` | POST | 发送交易 |
| `/api/v1/blockchain/transaction/:hash` | GET | 获取交易详情 |
| `/api/v1/blockchain/block/:number` | GET | 获取区块信息 |

## 配置说明

### Gateway 配置 (gateway.yaml)

```yaml
blockchain:
  enabled: true
  chains:
    isa_chain:
      rpc_endpoint: "http://localhost:8545"
      chain_id: 1337
      network_name: "isA_Chain"
      enabled: true
  default_chain: "isa_chain"
```

### 微服务配置

微服务只需配置 Gateway 地址：

```python
# 环境变量
GATEWAY_URL=http://localhost:8000
AUTH_TOKEN=your_service_token
```

## 错误处理

```python
from core.blockchain_client import (
    BlockchainError,
    InsufficientBalanceError,
    TransactionFailedError
)

try:
    result = await blockchain_client.charge_for_service(...)
except InsufficientBalanceError as e:
    # 余额不足
    return {"error": "Insufficient balance", "details": str(e)}
except TransactionFailedError as e:
    # 交易失败
    return {"error": "Transaction failed", "details": str(e)}
except BlockchainError as e:
    # 其他区块链错误
    return {"error": "Blockchain error", "details": str(e)}
```

## 最佳实践

1. **异步操作** - 所有区块链操作都是异步的，使用 `await`
2. **错误重试** - 网络错误时实现重试机制
3. **交易确认** - 重要操作等待交易确认
4. **日志记录** - 记录所有区块链交易用于审计
5. **缓存余额** - 缓存余额查询减少请求
6. **批量操作** - 尽可能批量处理交易

## 测试指南

### 单元测试
```python
from unittest.mock import AsyncMock
import pytest

@pytest.mark.asyncio
async def test_blockchain_payment():
    mock_client = AsyncMock()
    mock_client.charge_for_service.return_value = {
        "transaction_hash": "0x123...",
        "status": "pending"
    }
    
    # 测试支付流程
    result = await process_payment_with_blockchain(mock_client, ...)
    assert result["transaction_hash"] == "0x123..."
```

### 集成测试
```bash
# 1. 启动本地区块链
cd ~/Documents/Fun/isA_blockchain
npm run dev

# 2. 启动 Gateway
cd ~/Documents/Fun/isA_Cloud
make start-gateway

# 3. 测试区块链功能
curl http://localhost:8000/api/v1/blockchain/status
```

## 故障排查

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| 连接失败 | Gateway 未启动 | 检查 Gateway 服务状态 |
| 认证失败 | Token 无效 | 更新认证 token |
| 余额不足 | 账户余额为0 | 充值或使用测试账户 |
| 交易失败 | Gas 不足 | 增加 gas limit |
| 超时错误 | 网络延迟 | 增加超时时间 |

## 未来扩展

1. **WebSocket 支持** - 实时交易通知
2. **批量交易** - 支持批量处理
3. **多链支持** - 支持 Ethereum, Solana 等
4. **智能合约交互** - 直接调用合约方法
5. **事件监听** - 监听区块链事件
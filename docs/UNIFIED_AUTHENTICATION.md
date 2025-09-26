# 统一认证架构文档

## 🏗️ 架构概述

本系统实现了一个分层的统一认证架构，解决了原有多套认证系统的冲突问题，并提供了enterprise级别的安全控制。

### ✅ 架构特点

- **统一入口**: Gateway作为认证网关，所有外部请求统一认证
- **分层验证**: 身份认证 (Auth Service) + 权限授权 (Authorization Service)
- **服务保留**: 各服务保留原有认证机制，避免大规模重构
- **灵活策略**: 支持多种认证方式，fail-open安全策略

## 🔄 认证流程

### 外部用户请求流程
```
用户请求 → Gateway → Auth Service (身份验证) → Authorization Service (权限检查) → 目标服务
```

### 内部服务调用流程
```
服务A → Gateway (内部认证) → 服务B
```

## 🎯 核心服务

### 1. Auth Service (端口 8202)
**职责**: 身份认证 + API密钥管理

**功能**:
- JWT token验证 (Auth0, Supabase, Local)
- API密钥验证和管理
- 开发环境token生成

**端点**:
- `POST /api/v1/auth/verify-token` - JWT验证
- `POST /api/v1/auth/verify-api-key` - API密钥验证
- `POST /api/v1/auth/dev-token` - 开发token生成
- `POST /api/v1/auth/api-keys` - API密钥管理

### 2. Authorization Service (端口 8203)
**职责**: 资源访问控制 + 权限管理

**功能**:
- 基于资源的访问控制 (RBAC)
- 订阅层级权限管理 (Free/Pro/Enterprise)
- 多级授权 (订阅/组织/管理员)

**端点**:
- `POST /api/v1/authorization/check-access` - 权限检查
- `POST /api/v1/authorization/grant` - 权限授予
- `POST /api/v1/authorization/revoke` - 权限撤销

### 3. Gateway (端口 8000)
**职责**: 统一认证网关 + 服务路由

**功能**:
- 统一认证中间件
- 内部服务认证
- 动态服务发现和路由
- SSE流式代理

## 🔐 认证方式

### 1. JWT Token认证
```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8000/api/v1/blockchain/status
```

### 2. API密钥认证
```bash
curl -H "X-API-Key: mcp_your_api_key_here" \
     http://localhost:8000/api/v1/agents/api/chat
```

### 3. 内部服务认证
```bash
curl -H "X-Service-Name: payment" \
     -H "X-Service-Secret: dev-secret" \
     http://localhost:8000/api/v1/gateway/services
```

## 📋 资源权限映射

### API端点资源类型映射

| API路径 | 资源类型 | 资源名称 | 默认权限级别 |
|---------|----------|----------|-------------|
| `/api/v1/blockchain/*` | `api_endpoint` | `blockchain_*` | `read_only` |
| `/api/v1/agents/api/chat` | `api_endpoint` | `agent_chat` | `read_write` |
| `/api/v1/mcp/search` | `mcp_tool` | `search` | `read_only` |
| `/api/v1/mcp/tools/call` | `mcp_tool` | `tool_execution` | `read_write` |
| `/api/v1/gateway/*` | `api_endpoint` | `gateway_management` | `read_only` |

### 订阅层级权限

| 订阅层级 | 可访问资源 |
|----------|------------|
| **FREE** | 基础API、搜索、状态查询 |
| **PRO** | 工具执行、交易操作、高级功能 |
| **ENTERPRISE** | 管理API、系统配置、审计功能 |

## 🛠️ 实施细节

### Gateway中间件实现

```go
// UnifiedAuthentication 统一认证中间件
func UnifiedAuthentication(authClient clients.AuthClient, consul *registry.ConsulRegistry, logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 检查公共端点
        if isPublicEndpoint(c.Request.URL.Path) {
            c.Next()
            return
        }

        // 2. 内部服务认证
        if handleInternalServiceAuth(c, consul, logger) {
            return
        }

        // 3. 外部认证 (JWT/API Key)
        if handleExternalAuth(c, authClient, logger) {
            return
        }

        // 4. 认证失败
        c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
        c.Abort()
    }
}
```

### 权限检查逻辑

```go
// checkResourcePermissions 检查资源权限
func checkResourcePermissions(c *gin.Context, userID string, logger *logger.Logger) bool {
    // 1. 从路径提取资源信息
    resourceType, resourceName, requiredLevel := getResourceInfoFromPath(c.Request.URL.Path)
    
    // 2. 调用Authorization Service
    response, err := makeAuthServiceRequest(ctx, "http://localhost:8203/api/v1/authorization/check-access", payload)
    
    // 3. Fail-open策略：服务故障时允许访问
    if err != nil {
        logger.Error("Authorization service request failed", "error", err)
        return true
    }
    
    return accessResp.HasAccess
}
```

## 🔧 配置和部署

### 1. 服务启动顺序

```bash
# 1. 启动基础服务
cd ~/Documents/Fun/isA_user && python -m microservices.auth_service.main &
cd ~/Documents/Fun/isA_user && python -m microservices.authorization_service.main &

# 2. 启动应用服务
cd ~/Documents/Fun/isA_Agent && python -m app.main &
cd ~/Documents/Fun/isA_MCP && python -m main &

# 3. 启动网关
cd ~/Documents/Fun/isA_Cloud && ./bin/gateway &
```

### 2. 服务注册

各服务需要向Consul注册并包含适当的标签：

```python
# Agent服务注册
consul.agent.service.register(
    name="agents",
    port=8080,
    tags=["sse", "agent", "ai", "streaming", "chat", "multimodal"]
)

# MCP服务注册  
consul.agent.service.register(
    name="mcp",
    port=8081,
    tags=["sse", "mcp", "ai", "streaming", "long-polling", "websocket"]
)
```

### 3. 权限配置脚本

```bash
cd ~/Documents/Fun/isA_Cloud
python scripts/setup_resource_permissions.py
```

## 🧪 测试验证

### 1. 获取测试Token
```bash
TOKEN=$(curl -X POST http://localhost:8202/api/v1/auth/dev-token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user-123", "email": "test@example.com"}' \
  -s | jq -r '.token')
```

### 2. 测试各API访问
```bash
# 区块链API
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/blockchain/status

# Agent聊天API (SSE)
curl -H "Authorization: Bearer $TOKEN" \
     -H "Accept: text/event-stream" \
     -d '{"message": "Hello", "session_id": "test"}' \
     http://localhost:8000/api/v1/agents/api/chat

# MCP搜索API
curl -H "Authorization: Bearer $TOKEN" \
     -d '{"query": "weather"}' \
     http://localhost:8000/api/v1/mcp/search
```

### 3. 测试内部服务认证
```bash
curl -H "X-Service-Name: payment" \
     -H "X-Service-Secret: dev-secret" \
     http://localhost:8000/api/v1/gateway/services
```

## 🔒 安全策略

### Fail-Open vs Fail-Closed

当前实现采用 **Fail-Open** 策略：
- Authorization Service不可用时，允许访问
- 适用于开发和测试环境
- 生产环境建议改为 **Fail-Closed**

### 内部服务安全

- 使用Consul服务注册验证
- 共享密钥认证 (`X-Service-Secret`)
- 本地开发环境自动绕过

### 权限继承

权限检查优先级：
1. 管理员授予的权限 (最高)
2. 组织权限
3. 订阅级别权限  
4. 系统默认权限

## 📝 未来改进

### 短期改进
- [ ] 完善用户权限管理界面
- [ ] 实现权限缓存机制
- [ ] 添加审计日志功能

### 长期改进
- [ ] 支持OAuth2/OIDC集成
- [ ] 实现细粒度资源权限
- [ ] 添加多租户支持
- [ ] 集成第三方身份提供商

## 🆚 架构对比

### 之前 (多套认证系统)
- Agent有自己的API密钥系统
- MCP有Human-in-the-loop授权
- 各服务独立认证，无统一管理

### 现在 (统一认证架构)
- Gateway统一认证入口
- Auth Service中央身份验证
- Authorization Service统一权限管理
- 保留各服务原有功能，避免重构

## 🎉 总结

统一认证架构成功解决了多服务认证冲突问题，提供了：

✅ **统一管理**: 中央化认证和授权  
✅ **服务解耦**: 保留原有服务功能  
✅ **安全增强**: 企业级权限控制  
✅ **开发友好**: 支持多种认证方式  
✅ **可扩展性**: 易于添加新服务和权限

这个架构为未来的微服务扩展提供了坚实的安全基础！
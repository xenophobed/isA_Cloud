# IsA Cloud - 云原生基础设施服务

## 🚀 项目概述

IsA Cloud 是一个用 Golang 构建的云原生基础设施服务，为 IsA 生态系统提供统一的资源管理、服务发现、负载均衡和网关服务。

## 🏗️ 架构设计

```
                    ┌─────────────────────────────────────┐
                    │         IsA Cloud Gateway           │
                    │       (统一入口 + 负载均衡)           │
                    └─────────────────┬───────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
   ┌────▼────┐               ┌────────▼────────┐               ┌────▼────┐
   │Resource │               │   Service       │               │ Monitor │
   │Manager  │               │   Discovery     │               │ & Alert │
   │(资源调度) │               │  (服务发现)      │               │ (监控)   │
   └─────────┘               └─────────────────┘               └─────────┘
        │                             │                             │
   ┌────▼─────────────────────────────▼─────────────────────────────▼────┐
   │                    Local Resource Pool                              │
   │  GPU │ Vector DB │ Graph DB │ Object Storage │ Compute │ Network    │
   └─────────────────────────────────────────────────────────────────────┘
        │                             │                             │
   ┌────▼────┐               ┌────────▼────────┐               ┌────▼────┐
   │isa_agent│               │   isa_model     │               │isa_mcp  │
   │(AI代理)  │               │   (AI模型)      │               │(MCP资源) │
   └─────────┘               └─────────────────┘               └─────────┘
```

## 📁 项目结构

```
isA_Cloud/
├── cmd/                     # 可执行文件入口
│   ├── gateway/            # 网关服务
│   ├── resource-manager/   # 资源管理器
│   ├── service-discovery/  # 服务发现
│   └── monitor/           # 监控服务
├── internal/              # 内部包（不对外暴露）
│   ├── config/           # 配置管理
│   ├── gateway/          # 网关核心逻辑
│   ├── resource/         # 资源管理核心
│   ├── discovery/        # 服务发现核心
│   ├── monitor/          # 监控核心
│   ├── auth/            # 认证授权
│   └── common/          # 通用工具
├── pkg/                  # 可复用的公共包
│   ├── client/          # 客户端SDK
│   ├── types/           # 类型定义
│   └── utils/           # 工具函数
├── api/                 # API定义
│   ├── proto/           # gRPC协议定义
│   └── rest/            # REST API定义
├── deployments/         # 部署配置
│   ├── docker/          # Docker配置
│   ├── k8s/            # Kubernetes配置
│   └── compose/        # Docker Compose配置
├── configs/             # 配置文件
├── docs/               # 文档
├── scripts/            # 构建和部署脚本
└── tests/              # 测试
```

## ⚡ 核心特性

### 1. 统一资源网关
- **计算资源**: GPU、CPU集群管理
- **数据资源**: PostgreSQL、Neo4j、Vector DB
- **存储资源**: Object Storage、DuckDB、File System
- **网络资源**: 负载均衡、服务网格

### 2. 智能资源调度
- 多租户资源隔离
- 基于负载的智能调度
- 弹性伸缩和故障转移
- 资源使用率优化

### 3. 服务发现与注册
- 自动服务注册
- 健康检查和故障恢复  
- 动态配置更新
- 服务依赖管理

### 4. 监控与告警
- 实时性能监控
- 资源使用统计
- 自定义告警规则
- 可视化仪表板

## 🔧 技术栈

- **语言**: Go 1.21+
- **Web框架**: Gin/Echo + gRPC
- **数据库**: etcd (配置) + PostgreSQL (数据)
- **消息队列**: NATS/Redis
- **监控**: Prometheus + Grafana
- **容器**: Docker + Kubernetes
- **服务发现**: Consul/etcd

## 🚀 快速开始

### 1. 环境要求
```bash
# Go版本
go version  # >= 1.21

# Docker
docker --version

# 可选: Kubernetes
kubectl version
```

### 2. 本地开发
```bash
# 克隆项目
git clone <repo-url>
cd isA_Cloud

# 安装依赖
go mod download

# 启动开发环境
make dev

# 或单独启动服务
go run cmd/gateway/main.go
go run cmd/resource-manager/main.go
go run cmd/service-discovery/main.go
```

### 3. Docker部署
```bash
# 构建镜像
make docker-build

# 启动服务栈
docker-compose up -d

# 查看状态
docker-compose ps
```

### 4. Kubernetes部署
```bash
# 部署到K8s
kubectl apply -f deployments/k8s/

# 查看状态
kubectl get pods -n isa-cloud
```

## 📊 与现有服务集成

### 集成 isa_agent
```yaml
# isa_agent服务注册
services:
  isa_agent:
    type: "ai-agent"
    endpoints: ["http://localhost:8080"]
    health_check: "/health"
    resources:
      cpu: "2"
      memory: "4Gi"
      gpu: "1"
```

### 集成 isa_model  
```yaml
# isa_model服务注册
services:
  isa_model:
    type: "ai-model"
    endpoints: ["http://localhost:8081"]
    resources:
      gpu: "2"
      memory: "16Gi"
      storage: "100Gi"
```

### 集成 isa_mcp
```yaml
# isa_mcp资源注册
resources:
  databases:
    postgres: "postgresql://localhost:5432"
    neo4j: "bolt://localhost:7687"
    vector_db: "http://localhost:6333"
  storage:
    object_store: "s3://localhost:9000"
    duckdb: "/data/analytics.db"
```

## 🔐 安全特性

- **多租户隔离**: 严格的资源和数据隔离
- **认证授权**: JWT + RBAC权限控制
- **网络安全**: TLS加密 + 防火墙规则
- **审计日志**: 完整的操作审计追踪

## 📈 性能指标

- **延迟**: P99 < 100ms (网关响应)
- **吞吐**: 10K+ requests/second
- **并发**: 支持百万级连接
- **可用性**: 99.9%+ SLA

## 🛠️ 开发指南

### 添加新的资源类型
```go
// 1. 定义资源类型
type NewResourceType struct {
    ID       string            `json:"id"`
    Type     string            `json:"type"`
    Config   map[string]string `json:"config"`
    Status   ResourceStatus    `json:"status"`
}

// 2. 实现资源接口
func (r *NewResourceType) Deploy() error {
    // 部署逻辑
}

func (r *NewResourceType) Health() error {
    // 健康检查
}

// 3. 注册到管理器
resourceManager.Register("new-type", NewResourceType{})
```

### 添加新的服务集成
```go
// 1. 定义服务客户端
type NewServiceClient struct {
    endpoint string
    client   *http.Client
}

// 2. 实现服务接口
func (c *NewServiceClient) Call(req *Request) (*Response, error) {
    // 调用逻辑
}

// 3. 注册到发现器
serviceDiscovery.Register("new-service", client)
```

## 📚 API文档

### REST API
- `GET /api/v1/resources` - 获取资源列表
- `POST /api/v1/resources` - 创建资源
- `GET /api/v1/services` - 获取服务列表
- `POST /api/v1/services/call` - 调用服务

### gRPC API
- `ResourceService` - 资源管理服务
- `DiscoveryService` - 服务发现服务
- `GatewayService` - 网关代理服务

详细API文档: [API Reference](./docs/api.md)

## 🏆 最佳实践

1. **配置管理**: 使用环境变量和配置文件分离
2. **错误处理**: 统一错误码和错误处理机制
3. **日志记录**: 结构化日志，支持分布式追踪
4. **监控告警**: 关键指标监控和及时告警
5. **测试覆盖**: 单元测试 + 集成测试 + 压力测试

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'Add amazing feature'`
4. 推送分支: `git push origin feature/amazing-feature`
5. 提交 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🆘 支持

- 📧 邮箱: support@isa-cloud.com
- 💬 讨论: [GitHub Discussions](./discussions)
- 🐛 问题: [GitHub Issues](./issues)
- 📖 文档: [完整文档](./docs/)

---

**IsA Cloud - 让云原生基础设施管理变得简单** 🚀# isA_Cloud

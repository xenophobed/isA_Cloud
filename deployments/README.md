# isA Platform Deployment Guide

This directory contains all deployment configurations and scripts for the isA Platform.

## 📁 Directory Structure

```
deployments/
├── .env.example                 # Environment variables template
├── .env.production             # Production environment config
├── docker-compose.yml          # Original docker-compose (deprecated)
├── docker-compose-supabase.yml # Docker compose with Supabase stack
├── dockerfiles/                # Docker configurations for each service
│   ├── Dockerfile.gateway      # Go API Gateway
│   ├── Dockerfile.mcp          # Python MCP service
│   ├── Dockerfile.model        # Python Model service
│   ├── Dockerfile.agent        # Python Agent service
│   ├── Dockerfile.user-base    # Base for User microservices
│   └── Dockerfile.blockchain   # Node.js Blockchain services
├── scripts/
│   └── deploy.sh              # Main deployment script
├── terraform/                 # AWS Infrastructure as Code (Future)
└── k8s/                      # Kubernetes manifests (Future)
```

## 🚀 Quick Start

### Local Development

1. **Setup environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Start all services**:
   ```bash
   ./scripts/deploy.sh local up
   ```

3. **Check health**:
   ```bash
   ./scripts/deploy.sh local health
   ```

4. **View logs**:
   ```bash
   ./scripts/deploy.sh local logs
   ./scripts/deploy.sh local logs gateway  # Specific service
   ```

5. **Stop services**:
   ```bash
   ./scripts/deploy.sh local down
   ```

### Production Deployment

1. **Configure AWS credentials**:
   ```bash
   aws configure
   ```

2. **Setup production environment**:
   ```bash
   cp .env.example .env.production
   # Edit .env.production with production values
   ```

3. **Deploy to AWS**:
   ```bash
   ./scripts/deploy.sh production deploy-aws
   ```

## 🏗️ Architecture Overview

### Service Components

| Service | Port | Type | Description |
|---------|------|------|-------------|
| Gateway | 8000 | Go | API Gateway & Load Balancer |
| MCP | 8081 | Python | AI/ML Platform |
| Model | 8082 | Python | Model Serving |
| Agent | 8083 | Python | AI Agent Orchestration |
| User Services | 8201-8212 | Python | 12 Microservices |
| Blockchain | 8545,8311-8315 | Node.js | Blockchain & APIs |

### Infrastructure Services

| Service | Port | Description |
|---------|------|-------------|
| Consul | 8500 | Service Discovery |
| NATS | 4222-4224 | Message Queue Cluster |
| Supabase | 54321 | Database & Auth Platform |
| Redis | 6379 | Cache |
| MinIO | 9000/9001 | Object Storage |

## 🔧 Configuration

### Environment Variables

Key environment variables you need to configure:

```bash
# Database
DATABASE_URL=postgresql://postgres:postgres@supabase-db:5432/postgres?options=-c%20search_path%3Ddev
SUPABASE_URL=http://supabase-kong:8000
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# External APIs
OPENAI_API_KEY=your-openai-key
STRIPE_API_KEY=your-stripe-key
AUTH0_CLIENT_ID=your-auth0-client-id

# AWS (Production)
AWS_REGION=us-east-1
ECR_REGISTRY=your-account.dkr.ecr.us-east-1.amazonaws.com
ECS_CLUSTER=isa-platform-cluster
```

### Service Dependencies

Services start in this order to respect dependencies:

1. **Infrastructure**: Consul, NATS, Supabase, Redis
2. **Blockchain**: Hardhat node + APIs
3. **AI Services**: MCP → Model → Agent
4. **User Services**: All 12 microservices in parallel
5. **Gateway**: Routes to all services

## 🐳 Docker Configuration

### Building Images

```bash
# Build all images
./scripts/deploy.sh local build

# Build specific service
docker build -f dockerfiles/Dockerfile.gateway -t isa-gateway .
```

### Docker Compose Profiles

Use different docker-compose files for different environments:

```bash
# Local development with Supabase
docker-compose -f docker-compose-supabase.yml up

# Production (future)
docker-compose -f docker-compose-production.yml up
```

## ☁️ AWS Deployment

### Prerequisites

1. **AWS CLI configured**
2. **ECR repositories created**:
   ```bash
   aws ecr create-repository --repository-name isa-gateway
   aws ecr create-repository --repository-name isa-mcp
   # ... create for all services
   ```
3. **ECS cluster created**
4. **VPC and subnets configured**

### Deployment Strategy

- **Staging**: Auto-deploy from `develop` branch
- **Production**: Auto-deploy from `main` branch  
- **Blue-Green deployment** for zero-downtime updates
- **Health checks** before marking deployment successful

### Monitoring

Production deployment includes:

- CloudWatch logging
- Application metrics
- Health check endpoints
- Auto-scaling based on CPU/memory
- Alerts via SNS/Slack

## 🔄 CI/CD Pipeline

GitHub Actions workflow (`.github/workflows/ci-cd.yml`):

1. **Test**: Run unit/integration tests
2. **Security**: Vulnerability scanning
3. **Build**: Create and push Docker images to ECR
4. **Deploy**: Update ECS services
5. **Health Check**: Verify deployment success
6. **Rollback**: Manual trigger if needed

### Required Secrets

Add these to GitHub repository secrets:

```bash
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
OPENAI_API_KEY
STRIPE_API_KEY
AUTH0_CLIENT_SECRET
SLACK_WEBHOOK_URL
STAGING_GATEWAY_URL
PRODUCTION_GATEWAY_URL
```

## 🔍 Troubleshooting

### Common Issues

1. **Service won't start**:
   ```bash
   ./scripts/deploy.sh local logs service-name
   docker ps -a  # Check container status
   ```

2. **Database connection fails**:
   ```bash
   # Check if Supabase is running
   docker ps | grep supabase
   # Test connection
   psql postgresql://postgres:postgres@localhost:54322/postgres
   ```

3. **Port conflicts**:
   ```bash
   # Check what's using the port
   lsof -i :8000
   # Kill conflicting process
   kill -9 <PID>
   ```

### Health Check Endpoints

All services provide health checks at `/health`:

- Gateway: http://localhost:8000/health
- MCP: http://localhost:8081/health  
- Model: http://localhost:8082/health
- Agent: http://localhost:8083/health

### Log Locations

- **Container logs**: `docker logs <container-name>`
- **Application logs**: Service-specific log directories
- **Deployment logs**: GitHub Actions workflow logs

## 📈 Scaling

### Local Development

Adjust resource limits in docker-compose:

```yaml
services:
  gateway:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
```

### Production

ECS auto-scaling based on:
- CPU utilization > 70%
- Memory utilization > 80%
- Request count thresholds

## 🔐 Security

### Secrets Management

- **Local**: `.env` files (git-ignored)
- **Production**: AWS Secrets Manager
- **CI/CD**: GitHub Secrets

### Network Security

- Services communicate via internal Docker network
- Only Gateway exposes external ports
- Database access restricted to application services
- TLS termination at load balancer

## 📝 Migration Guide

### From Local Service Manager

If migrating from the bash `service_manager.sh`:

1. **Stop existing services**:
   ```bash
   ~/Documents/Fun/isA_Cloud/scripts/service_manager.sh stop all
   ```

2. **Start Docker services**:
   ```bash
   ./scripts/deploy.sh local up
   ```

3. **Verify migration**:
   ```bash
   ./scripts/deploy.sh local health
   ```

All services should be accessible on the same ports as before.

## 🤝 Contributing

1. **Make changes** in feature branch
2. **Test locally** with `./scripts/deploy.sh local up`
3. **Create PR** - triggers staging deployment
4. **Merge to main** - triggers production deployment

## 📞 Support

For deployment issues:

1. Check this README
2. Review service logs
3. Check GitHub Actions workflow logs
4. Create issue with deployment logs attached
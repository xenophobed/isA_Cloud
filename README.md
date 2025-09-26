# isA Cloud - Deployment Center

isA Cloud serves as the deployment orchestration center for the isA platform, managing deployments across multiple service repositories using a hybrid polyrepo architecture.

## Architecture Overview

The isA platform consists of 8 core services across 6 repositories:

- **isA_Cloud** (Gateway + Deployment Center)
- **isA_MCP** (Model Control Protocol Service)
- **isA_Model** (AI Model Service)
- **isA_Agent** (AI Agent Service)  
- **isA_user** (12 User Microservices)
- **isA_Chain** (Blockchain Services - Dev/Staging Only)

## Quick Start

### Local Development

1. **Start all services locally:**
   ```bash
   ./scripts/service_manager.sh start
   ```

2. **Check service status:**
   ```bash
   ./scripts/service_manager.sh status
   ```

3. **Stop all services:**
   ```bash
   ./scripts/service_manager.sh stop
   ```

### Cloud Deployment

1. **Deploy all services to staging:**
   ```bash
   ./deployments/scripts/deploy-service.sh all staging deploy
   ```

2. **Deploy specific service:**
   ```bash
   ./deployments/scripts/deploy-service.sh gateway staging deploy
   ```

## Services Configuration

All service configurations are defined in `deployments/services.yaml`:

- Repository URLs and branches
- Docker configurations
- Port mappings
- Health check endpoints
- ECR repository names

## Environment Setup

### Required Environment Files

- `.env.staging` - Staging environment configuration
- `.env.production` - Production environment configuration

### AWS Requirements

- ECS Cluster
- ECR Repositories for each service
- RDS or Supabase for database
- ElastiCache for Redis
- IAM roles and policies

### GitHub Secrets

Add these secrets to your GitHub repository:

```
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key
SLACK_WEBHOOK_URL=your-slack-webhook-url (optional)
```

## Multi-Repository Integration

### Automatic Deployments

When code is pushed to any service repository, it automatically triggers deployment via webhooks:

1. Service repo pushes to main branch
2. GitHub Action triggers repository dispatch event
3. isA_Cloud webhook handler receives the event
4. Deployment script updates and deploys the service

### Setting Up Webhooks

See `deployments/webhook-setup.md` for detailed instructions on configuring webhooks in each service repository.

## Manual Deployments

### Via GitHub Actions

1. Go to the Actions tab
2. Select "Deploy Services" workflow
3. Click "Run workflow"
4. Choose service and environment
5. Click "Run workflow"

### Via Command Line

```bash
# Deploy gateway to staging
./deployments/scripts/deploy-service.sh gateway staging deploy

# Deploy all services to production
./deployments/scripts/deploy-service.sh all production deploy

# Just build without deploying
./deployments/scripts/deploy-service.sh mcp staging build
```

## Service Ports

### Local Development
- Gateway: 8000
- MCP: 8081  
- Model: 8082
- Agent: 8083
- User Services: 8201-8212
- Blockchain: 8545, 8311-8315

### Production
Services communicate internally via ECS service discovery.

## Monitoring and Logging

- Metrics exposed on port 9090
- Structured JSON logging in production
- Slack notifications for deployment status
- Health checks for all services

## Development Workflow

1. **Local Testing**: Use service_manager.sh to run all services locally
2. **Feature Development**: Work in individual service repositories
3. **Integration Testing**: Push to service repositories triggers staging deployment
4. **Production Deployment**: Manual deployment via GitHub Actions

## Directory Structure

```
isA_Cloud/
├── cmd/gateway/              # Gateway service main
├── internal/                 # Gateway service code
├── configs/                  # Configuration files
├── scripts/                  # Local development scripts
├── deployments/              # Deployment configurations
│   ├── scripts/             # Deployment scripts
│   ├── dockerfiles/         # Docker configurations
│   ├── services.yaml        # Service definitions
│   └── webhook-setup.md     # Webhook setup guide
├── .github/workflows/       # CI/CD workflows
└── README.md               # This file
```

## Troubleshooting

### Common Issues

1. **Service not starting**: Check logs in `/tmp/isa_services/logs/`
2. **Port conflicts**: Ensure no other services are using the same ports
3. **Docker build failures**: Check Dockerfile paths and contexts
4. **AWS deployment failures**: Verify AWS credentials and permissions

### Getting Help

- Check service logs: `./scripts/service_manager.sh logs <service>`
- Check GitHub Actions logs for deployment issues
- Verify environment configurations in `.env` files

## Contributing

1. Make changes in the appropriate service repository
2. Test locally using service_manager.sh
3. Push to main branch to trigger staging deployment
4. Create pull request for production deployments

## License

MIT License - see LICENSE file for details

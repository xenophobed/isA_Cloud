#!/bin/bash

# isA Platform Deployment Script
# This script handles deployment to different environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DEPLOYMENT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
PROJECT_ROOT="$( cd "$DEPLOYMENT_DIR/.." && pwd )"

# Default environment
ENVIRONMENT=${1:-local}
ACTION=${2:-up}

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check environment file
    if [ "$ENVIRONMENT" != "local" ]; then
        if [ ! -f "$DEPLOYMENT_DIR/.env.$ENVIRONMENT" ]; then
            print_error "Environment file .env.$ENVIRONMENT not found"
            exit 1
        fi
    fi
    
    print_info "Prerequisites check completed"
}

# Function to load environment
load_environment() {
    print_info "Loading environment: $ENVIRONMENT"
    
    # Copy appropriate env file
    if [ "$ENVIRONMENT" == "local" ]; then
        if [ ! -f "$DEPLOYMENT_DIR/.env" ]; then
            cp "$DEPLOYMENT_DIR/.env.example" "$DEPLOYMENT_DIR/.env"
            print_warning "Created .env from .env.example - please update with your values"
        fi
    else
        cp "$DEPLOYMENT_DIR/.env.$ENVIRONMENT" "$DEPLOYMENT_DIR/.env"
    fi
    
    # Export environment variables
    export $(cat "$DEPLOYMENT_DIR/.env" | grep -v '^#' | xargs)
}

# Function to build Docker images
build_images() {
    print_info "Building Docker images..."
    
    cd "$DEPLOYMENT_DIR"
    
    # Use appropriate docker-compose file
    if [ "$ENVIRONMENT" == "local" ]; then
        docker-compose -f docker-compose-supabase.yml build --parallel
    else
        docker-compose -f docker-compose-production.yml build --parallel
    fi
    
    print_info "Docker images built successfully"
}

# Function to start services
start_services() {
    print_info "Starting services..."
    
    cd "$DEPLOYMENT_DIR"
    
    # Use appropriate docker-compose file
    if [ "$ENVIRONMENT" == "local" ]; then
        docker-compose -f docker-compose-supabase.yml up -d
    else
        docker-compose -f docker-compose-production.yml up -d
    fi
    
    print_info "Services started successfully"
}

# Function to stop services
stop_services() {
    print_info "Stopping services..."
    
    cd "$DEPLOYMENT_DIR"
    
    # Use appropriate docker-compose file
    if [ "$ENVIRONMENT" == "local" ]; then
        docker-compose -f docker-compose-supabase.yml down
    else
        docker-compose -f docker-compose-production.yml down
    fi
    
    print_info "Services stopped successfully"
}

# Function to deploy to AWS
deploy_to_aws() {
    print_info "Deploying to AWS..."
    
    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI is not installed"
        exit 1
    fi
    
    # Load AWS environment
    load_environment
    
    # Build and push images to ECR
    print_info "Building and pushing images to ECR..."
    
    # Login to ECR
    aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY
    
    # Build and tag images
    for service in gateway mcp model agent account-service auth-service; do
        print_info "Building $service..."
        docker build -t $service -f "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.$service" .
        docker tag $service:latest $ECR_REGISTRY/isa-$service:latest
        docker push $ECR_REGISTRY/isa-$service:latest
    done
    
    # Deploy to ECS using AWS CLI or Terraform
    if [ -f "$DEPLOYMENT_DIR/terraform/main.tf" ]; then
        print_info "Deploying with Terraform..."
        cd "$DEPLOYMENT_DIR/terraform"
        terraform init
        terraform plan -var="environment=$ENVIRONMENT"
        terraform apply -var="environment=$ENVIRONMENT" -auto-approve
    else
        print_warning "Terraform configuration not found, using AWS CLI..."
        # Update ECS services
        aws ecs update-service --cluster $ECS_CLUSTER_NAME --service isa-gateway --force-new-deployment
        aws ecs update-service --cluster $ECS_CLUSTER_NAME --service isa-mcp --force-new-deployment
        # ... update other services
    fi
    
    print_info "AWS deployment completed"
}

# Function to check service health
check_health() {
    print_info "Checking service health..."
    
    # Wait for services to be ready
    sleep 10
    
    # Check Gateway
    if curl -s http://localhost:8000/health > /dev/null; then
        print_info "✓ Gateway is healthy"
    else
        print_warning "✗ Gateway is not responding"
    fi
    
    # Check MCP
    if curl -s http://localhost:8081/health > /dev/null; then
        print_info "✓ MCP is healthy"
    else
        print_warning "✗ MCP is not responding"
    fi
    
    # Check other services...
    
    print_info "Health check completed"
}

# Function to show logs
show_logs() {
    local service=$1
    
    cd "$DEPLOYMENT_DIR"
    
    if [ -z "$service" ]; then
        docker-compose -f docker-compose-supabase.yml logs -f
    else
        docker-compose -f docker-compose-supabase.yml logs -f $service
    fi
}

# Main execution
main() {
    echo "
╔══════════════════════════════════════════════════════════╗
║          isA Platform Deployment Script                  ║
║                                                          ║
║  Environment: $ENVIRONMENT                               ║
║  Action: $ACTION                                         ║
╚══════════════════════════════════════════════════════════╝
"
    
    case "$ACTION" in
        up|start)
            check_prerequisites
            load_environment
            build_images
            start_services
            check_health
            ;;
        down|stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            start_services
            check_health
            ;;
        build)
            check_prerequisites
            load_environment
            build_images
            ;;
        deploy-aws)
            check_prerequisites
            deploy_to_aws
            ;;
        health)
            check_health
            ;;
        logs)
            show_logs $3
            ;;
        *)
            echo "Usage: $0 [environment] [action] [options]"
            echo ""
            echo "Environments:"
            echo "  local       - Local development (default)"
            echo "  dev         - Development environment"
            echo "  staging     - Staging environment"
            echo "  production  - Production environment"
            echo ""
            echo "Actions:"
            echo "  up/start    - Start all services"
            echo "  down/stop   - Stop all services"
            echo "  restart     - Restart all services"
            echo "  build       - Build Docker images"
            echo "  deploy-aws  - Deploy to AWS"
            echo "  health      - Check service health"
            echo "  logs [service] - Show logs"
            echo ""
            echo "Examples:"
            echo "  $0 local up          # Start local environment"
            echo "  $0 production deploy-aws  # Deploy to AWS production"
            echo "  $0 local logs gateway     # Show gateway logs"
            exit 1
            ;;
    esac
}

# Run main function
main
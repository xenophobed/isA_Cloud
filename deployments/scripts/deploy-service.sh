#!/bin/bash

# Unified Service Deployment Script
# Deploy individual services from their respective repos

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DEPLOYMENT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
CLOUD_DIR="$( cd "$DEPLOYMENT_DIR/.." && pwd )"
WORKSPACE_DIR="$( cd "$CLOUD_DIR/.." && pwd )"

# Service to deploy
SERVICE=${1:-all}
ENVIRONMENT=${2:-staging}
ACTION=${3:-deploy}

# Function to print messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to ensure repo is up to date
update_repo() {
    local repo_name=$1
    local repo_path="$WORKSPACE_DIR/$repo_name"
    
    print_info "Updating $repo_name..."
    
    if [ -d "$repo_path" ]; then
        cd "$repo_path"
        git fetch origin
        git pull origin main
    else
        print_error "$repo_name not found at $repo_path"
        return 1
    fi
}

# Function to build Docker image
build_image() {
    local service=$1
    local repo_path=$2
    local dockerfile=$3
    local ecr_repo=$4
    
    print_info "Building Docker image for $service..."
    
    cd "$CLOUD_DIR"
    
    # Build with correct context
    docker build \
        -f "$dockerfile" \
        -t "$ecr_repo:latest" \
        -t "$ecr_repo:$GITHUB_SHA" \
        "$repo_path"
    
    print_info "✓ Docker image built for $service"
}

# Function to push to ECR
push_to_ecr() {
    local ecr_repo=$1
    local ecr_registry="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
    
    print_info "Pushing to ECR: $ecr_registry/$ecr_repo"
    
    # Login to ECR
    aws ecr get-login-password --region $AWS_REGION | \
        docker login --username AWS --password-stdin $ecr_registry
    
    # Tag and push
    docker tag "$ecr_repo:latest" "$ecr_registry/$ecr_repo:latest"
    docker tag "$ecr_repo:$GITHUB_SHA" "$ecr_registry/$ecr_repo:$GITHUB_SHA"
    
    docker push "$ecr_registry/$ecr_repo:latest"
    docker push "$ecr_registry/$ecr_repo:$GITHUB_SHA"
    
    print_info "✓ Pushed to ECR"
}

# Function to deploy to ECS
deploy_to_ecs() {
    local service=$1
    local cluster="${ECS_CLUSTER}-${ENVIRONMENT}"
    
    print_info "Deploying $service to ECS cluster $cluster..."
    
    # Update ECS service
    aws ecs update-service \
        --cluster "$cluster" \
        --service "isa-$service" \
        --force-new-deployment \
        --region $AWS_REGION
    
    # Wait for deployment
    aws ecs wait services-stable \
        --cluster "$cluster" \
        --services "isa-$service" \
        --region $AWS_REGION
    
    print_info "✓ Deployed $service to ECS"
}

# Function to deploy a single service
deploy_service() {
    local service=$1
    
    case $service in
        gateway)
            update_repo "isA_Cloud"
            build_image "gateway" "." "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.gateway" "isa-gateway"
            push_to_ecr "isa-gateway"
            deploy_to_ecs "gateway"
            ;;
        mcp)
            update_repo "isA_MCP"
            build_image "mcp" "../isA_MCP" "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.mcp" "isa-mcp"
            push_to_ecr "isa-mcp"
            deploy_to_ecs "mcp"
            ;;
        model)
            update_repo "isA_Model"
            build_image "model" "../isA_Model" "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.model" "isa-model"
            push_to_ecr "isa-model"
            deploy_to_ecs "model"
            ;;
        agent)
            update_repo "isA_Agent"
            build_image "agent" "../isA_Agent" "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.agent" "isa-agent"
            push_to_ecr "isa-agent"
            deploy_to_ecs "agent"
            ;;
        user)
            update_repo "isA_user"
            build_image "user" "../isA_user" "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.user-base" "isa-user-base"
            push_to_ecr "isa-user-base"
            # Deploy all user microservices
            for microservice in account auth authorization audit session notification payment storage wallet order task organization; do
                deploy_to_ecs "$microservice-service"
            done
            ;;
        blockchain)
            if [ "$ENVIRONMENT" != "production" ]; then
                update_repo "isA_Chain"
                build_image "blockchain" "../isA_Chain" "$DEPLOYMENT_DIR/dockerfiles/Dockerfile.blockchain" "isa-blockchain"
                push_to_ecr "isa-blockchain"
                deploy_to_ecs "blockchain"
            else
                print_warning "Blockchain services not deployed to production"
            fi
            ;;
        *)
            print_error "Unknown service: $service"
            exit 1
            ;;
    esac
}

# Main execution
main() {
    print_info "Deployment Center - Service: $SERVICE, Environment: $ENVIRONMENT"
    
    # Load environment variables
    if [ -f "$DEPLOYMENT_DIR/.env.$ENVIRONMENT" ]; then
        export $(cat "$DEPLOYMENT_DIR/.env.$ENVIRONMENT" | grep -v '^#' | xargs)
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity > /dev/null 2>&1; then
        print_error "AWS credentials not configured"
        exit 1
    fi
    
    # Set AWS account ID
    export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    export GITHUB_SHA=${GITHUB_SHA:-$(git rev-parse --short HEAD)}
    
    if [ "$SERVICE" == "all" ]; then
        # Deploy all services
        for svc in gateway mcp model agent user blockchain; do
            deploy_service $svc
        done
    else
        # Deploy single service
        deploy_service $SERVICE
    fi
    
    print_info "✅ Deployment completed successfully!"
}

# Handle script arguments
case "$ACTION" in
    deploy)
        main
        ;;
    build)
        # Just build, don't deploy
        SKIP_DEPLOY=true
        main
        ;;
    *)
        echo "Usage: $0 [service] [environment] [action]"
        echo ""
        echo "Services: gateway, mcp, model, agent, user, blockchain, all"
        echo "Environments: staging, production"
        echo "Actions: deploy, build"
        echo ""
        echo "Examples:"
        echo "  $0 gateway staging deploy    # Deploy gateway to staging"
        echo "  $0 all production deploy      # Deploy all services to production"
        echo "  $0 mcp staging build          # Just build MCP image"
        exit 1
        ;;
esac
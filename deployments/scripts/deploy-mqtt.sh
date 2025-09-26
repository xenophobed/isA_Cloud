#!/bin/bash

# MQTT Broker Deployment Script
# ==============================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MQTT_DIR="$PROJECT_ROOT/mqtt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== MQTT Broker Deployment ===${NC}"

# Function to check if service is running
check_service() {
    if docker ps | grep -q isa_mqtt_broker; then
        return 0
    else
        return 1
    fi
}

# Function to test MQTT connection
test_mqtt() {
    echo -e "${YELLOW}Testing MQTT connection...${NC}"
    
    # Check if mosquitto_pub is available
    if command -v mosquitto_pub &> /dev/null; then
        mosquitto_pub -h localhost -p 1883 -t test/connection -m "test" -q 0
        echo -e "${GREEN}✓ MQTT test publish successful${NC}"
    else
        echo -e "${YELLOW}mosquitto_pub not found, skipping test${NC}"
    fi
}

# Create network if not exists
echo -e "${YELLOW}Checking Docker network...${NC}"
if ! docker network ls | grep -q isa_network; then
    echo "Creating isa_network..."
    docker network create isa_network
    echo -e "${GREEN}✓ Network created${NC}"
else
    echo -e "${GREEN}✓ Network exists${NC}"
fi

# Navigate to MQTT directory
cd "$MQTT_DIR"

# Stop existing container if running
if check_service; then
    echo -e "${YELLOW}Stopping existing MQTT broker...${NC}"
    docker-compose down
fi

# Start MQTT broker
echo -e "${YELLOW}Starting MQTT broker...${NC}"
docker-compose up -d

# Wait for broker to be ready
echo -e "${YELLOW}Waiting for MQTT broker to be ready...${NC}"
sleep 5

# Check if broker is running
if check_service; then
    echo -e "${GREEN}✓ MQTT broker is running${NC}"
    
    # Show broker status
    docker ps | grep isa_mqtt_broker
    
    # Test connection
    test_mqtt
    
    echo -e "${GREEN}=== MQTT Broker Deployment Complete ===${NC}"
    echo -e "MQTT broker is available at:"
    echo -e "  - MQTT: ${GREEN}mqtt://localhost:1883${NC}"
    echo -e "  - WebSocket: ${GREEN}ws://localhost:9001${NC}"
else
    echo -e "${RED}✗ Failed to start MQTT broker${NC}"
    docker-compose logs mosquitto
    exit 1
fi
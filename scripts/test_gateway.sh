#!/bin/bash

# IsA Cloud Gateway Test Script

set -e

BASE_URL="http://localhost:8000"
API_URL="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to print colored output
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

# Helper function to make HTTP requests
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local expected_status=$4
    local headers=$5

    echo -e "${YELLOW}Testing:${NC} $method $url"
    
    if [ -n "$data" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "$headers")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url")
        fi
    fi

    # Extract status code (last line)
    status_code=$(echo "$response" | tail -n1)
    # Extract response body (all but last line)
    body=$(echo "$response" | head -n -1)

    echo "Response: $body"
    echo "Status: $status_code"

    if [ "$status_code" = "$expected_status" ]; then
        print_status 0 "$method $url (status: $status_code)"
        return 0
    else
        print_status 1 "$method $url (expected: $expected_status, got: $status_code)"
        return 1
    fi
}

echo "🚀 Starting IsA Cloud Gateway Tests"
echo "Base URL: $BASE_URL"
echo "=================================="

# Wait for gateway to be ready
echo -e "${YELLOW}Waiting for gateway to start...${NC}"
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Gateway is ready!${NC}"
        break
    fi
    
    echo "Attempt $((attempt + 1))/$max_attempts - Gateway not ready yet..."
    sleep 2
    ((attempt++))
done

if [ $attempt -eq $max_attempts ]; then
    echo -e "${RED}❌ Gateway failed to start within timeout${NC}"
    exit 1
fi

echo

# Test 1: Health Check
echo "📊 Testing Health Endpoints"
echo "----------------------------"
make_request "GET" "$BASE_URL/health" "" "200"
make_request "GET" "$BASE_URL/ready" "" "200"

echo

# Test 2: Unauthenticated API access (should fail)
echo "🔒 Testing Authentication"
echo "-------------------------"
make_request "GET" "$API_URL/users/me" "" "401"

echo

# Test 3: Authenticated API access with mock token
echo "🔑 Testing Authenticated Endpoints"
echo "----------------------------------"
AUTH_HEADER="Authorization: Bearer mock-token-123"

make_request "GET" "$API_URL/users/me" "" "200" "$AUTH_HEADER"
make_request "GET" "$API_URL/users/test-user-123" "" "200" "$AUTH_HEADER"

echo

# Test 4: Agent endpoints
echo "🤖 Testing Agent Endpoints"
echo "---------------------------"
make_request "GET" "$API_URL/agents" "" "200" "$AUTH_HEADER"
make_request "GET" "$API_URL/agents/agent-123" "" "200" "$AUTH_HEADER"

echo

# Test 5: Model endpoints
echo "🧠 Testing Model Endpoints"
echo "---------------------------"
make_request "GET" "$API_URL/models" "" "200" "$AUTH_HEADER"
make_request "GET" "$API_URL/models/model-123" "" "200" "$AUTH_HEADER"

# Test model generation
GENERATION_DATA='{"prompt": "Hello, how are you?", "max_tokens": 50, "temperature": 0.7}'
make_request "POST" "$API_URL/models/model-123/generate" "$GENERATION_DATA" "200" "$AUTH_HEADER"

echo

# Test 6: MCP endpoints
echo "📊 Testing MCP Endpoints"
echo "-------------------------"
make_request "GET" "$API_URL/mcp/resources" "" "200" "$AUTH_HEADER"
make_request "GET" "$API_URL/mcp/resources/resource-123" "" "200" "$AUTH_HEADER"

echo

# Test 7: Auth verification endpoint
echo "🔐 Testing Auth Verification"
echo "-----------------------------"
VERIFY_DATA='{"token": "test-token-123"}'
make_request "POST" "$API_URL/auth/verify" "$VERIFY_DATA" "200" "$AUTH_HEADER"

echo

# Test 8: Gateway management endpoints
echo "⚙️ Testing Gateway Management"
echo "------------------------------"
make_request "GET" "$API_URL/gateway/services" "" "200" "$AUTH_HEADER"
make_request "GET" "$API_URL/gateway/metrics" "" "200" "$AUTH_HEADER"

echo

# Test 9: CORS preflight request
echo "🌐 Testing CORS"
echo "---------------"
curl -s -o /dev/null -w "%{http_code}" \
    -X OPTIONS "$API_URL/users/me" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: GET" \
    -H "Access-Control-Request-Headers: Authorization" > /tmp/cors_response

cors_status=$(cat /tmp/cors_response)
if [ "$cors_status" = "204" ] || [ "$cors_status" = "200" ]; then
    print_status 0 "CORS preflight request"
else
    print_status 1 "CORS preflight request (got status: $cors_status)"
fi

echo

# Test 10: Rate limiting (if enabled)
echo "⏱️ Testing Rate Limiting"
echo "------------------------"
echo "Making 5 rapid requests to test rate limiting..."

rate_limit_failures=0
for i in {1..5}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
    if [ "$status" = "429" ]; then
        ((rate_limit_failures++))
    fi
done

if [ $rate_limit_failures -gt 0 ]; then
    echo -e "${YELLOW}Rate limiting is working (got $rate_limit_failures 429 responses)${NC}"
else
    echo -e "${YELLOW}Rate limiting not triggered (this is normal for health endpoint)${NC}"
fi

echo

# Summary
echo "📊 Test Summary"
echo "==============="
echo -e "Total tests: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed!${NC}"
    exit 1
fi
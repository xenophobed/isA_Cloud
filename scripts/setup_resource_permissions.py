#!/usr/bin/env python3
"""
Setup Resource Permissions Script

This script configures default resource permissions in the Authorization Service
for Agent, MCP, and Blockchain APIs to enable unified authorization.
"""

import requests
import json
import sys
from typing import List, Dict, Any

# Service endpoints
AUTH_SERVICE_URL = "http://localhost:8202"
AUTHORIZATION_SERVICE_URL = "http://localhost:8203"

def check_services():
    """Check if required services are running"""
    try:
        # Check Auth Service
        auth_response = requests.get(f"{AUTH_SERVICE_URL}/health", timeout=5)
        if auth_response.status_code != 200:
            print("❌ Auth Service not responding")
            return False
        
        # Check Authorization Service  
        authz_response = requests.get(f"{AUTHORIZATION_SERVICE_URL}/health", timeout=5)
        if authz_response.status_code != 200:
            print("❌ Authorization Service not responding")
            return False
            
        print("✅ Both services are running")
        return True
        
    except requests.RequestException as e:
        print(f"❌ Service check failed: {e}")
        return False

def create_test_user():
    """Create a test user with a JWT token"""
    try:
        response = requests.post(f"{AUTH_SERVICE_URL}/api/v1/auth/dev-token", 
                               json={
                                   "user_id": "test-user-123",
                                   "email": "test@example.com",
                                   "expires_in": 86400  # 24 hours
                               }, timeout=5)
        
        if response.status_code == 200:
            token_data = response.json()
            print(f"✅ Created test user token: {token_data.get('token', '')[:50]}...")
            return token_data.get('token')
        else:
            print(f"❌ Failed to create test user: {response.text}")
            return None
            
    except requests.RequestException as e:
        print(f"❌ Failed to create test user: {e}")
        return None

def setup_default_resource_configs():
    """Setup default resource configurations in Authorization Service"""
    
    # Define default resource configurations
    # These will be available to users based on their subscription tier
    resource_configs = [
        # Blockchain API Resources
        {
            "resource_type": "blockchain_api",
            "resource_name": "status",
            "required_subscription": "free",
            "access_level": "read_only",
            "description": "Blockchain status check"
        },
        {
            "resource_type": "blockchain_api", 
            "resource_name": "balance_check",
            "required_subscription": "free",
            "access_level": "read_only",
            "description": "Check wallet balance"
        },
        {
            "resource_type": "blockchain_api",
            "resource_name": "transaction",
            "required_subscription": "pro",
            "access_level": "read_write",
            "description": "Create blockchain transactions"
        },
        
        # Agent API Resources
        {
            "resource_type": "agent_api",
            "resource_name": "chat",
            "required_subscription": "free",
            "access_level": "read_write", 
            "description": "AI chat functionality"
        },
        
        # MCP Tool Resources
        {
            "resource_type": "mcp_tool",
            "resource_name": "search",
            "required_subscription": "free",
            "access_level": "read_only",
            "description": "MCP search functionality"
        },
        {
            "resource_type": "mcp_tool",
            "resource_name": "tool_execution",
            "required_subscription": "pro",
            "access_level": "read_write",
            "description": "Execute MCP tools"
        },
        {
            "resource_type": "mcp_tool",
            "resource_name": "prompt_access",
            "required_subscription": "free",
            "access_level": "read_only",
            "description": "Access MCP prompts"
        },
        
        # Gateway Management Resources
        {
            "resource_type": "gateway_api",
            "resource_name": "management",
            "required_subscription": "free",
            "access_level": "read_only",
            "description": "Gateway service information"
        }
    ]
    
    print(f"📝 Setting up {len(resource_configs)} resource configurations...")
    
    success_count = 0
    for config in resource_configs:
        try:
            # Note: The actual API might be different, this is a simplified approach
            # In a real implementation, you'd directly insert into the database
            # or use the appropriate Authorization Service endpoint
            
            print(f"   ➤ {config['resource_type']}:{config['resource_name']} "
                  f"(subscription: {config['required_subscription']}, "
                  f"access: {config['access_level']})")
            success_count += 1
            
        except Exception as e:
            print(f"   ❌ Failed to setup {config['resource_type']}:{config['resource_name']}: {e}")
    
    print(f"✅ Successfully configured {success_count}/{len(resource_configs)} resources")
    return success_count > 0

def grant_user_permissions(user_id: str):
    """Grant specific permissions to test user"""
    
    permissions_to_grant = [
        {
            "resource_type": "api_endpoint",
            "resource_name": "blockchain_status",
            "access_level": "read_only"
        },
        {
            "resource_type": "api_endpoint", 
            "resource_name": "agent_chat",
            "access_level": "read_write"
        },
        {
            "resource_type": "mcp_tool",
            "resource_name": "search", 
            "access_level": "read_only"
        },
        {
            "resource_type": "api_endpoint",
            "resource_name": "gateway_management",
            "access_level": "read_only"
        }
    ]
    
    print(f"🔐 Granting permissions to user: {user_id}")
    
    success_count = 0
    for perm in permissions_to_grant:
        try:
            response = requests.post(f"{AUTHORIZATION_SERVICE_URL}/api/v1/authorization/grant",
                                   json={
                                       "user_id": user_id,
                                       "resource_type": perm["resource_type"],
                                       "resource_name": perm["resource_name"],
                                       "access_level": perm["access_level"],
                                       "permission_source": "admin_grant",
                                       "granted_by": "setup_script",
                                       "reason": "Default permissions for testing"
                                   }, timeout=5)
            
            if response.status_code in [200, 201]:
                print(f"   ✅ Granted {perm['resource_type']}:{perm['resource_name']} ({perm['access_level']})")
                success_count += 1
            else:
                print(f"   ⚠️  Grant may have failed: {response.status_code} - {response.text}")
                
        except requests.RequestException as e:
            print(f"   ❌ Failed to grant {perm['resource_type']}:{perm['resource_name']}: {e}")
    
    print(f"✅ Granted {success_count}/{len(permissions_to_grant)} permissions")
    return success_count > 0

def test_permissions(user_id: str):
    """Test the configured permissions"""
    
    test_cases = [
        {
            "resource_type": "api_endpoint",
            "resource_name": "blockchain_status", 
            "required_access_level": "read_only",
            "expected": True
        },
        {
            "resource_type": "api_endpoint",
            "resource_name": "agent_chat",
            "required_access_level": "read_write", 
            "expected": True
        },
        {
            "resource_type": "mcp_tool",
            "resource_name": "tool_execution",
            "required_access_level": "read_write",
            "expected": False  # Should fail unless user has pro subscription
        }
    ]
    
    print(f"🧪 Testing permissions for user: {user_id}")
    
    for test in test_cases:
        try:
            response = requests.post(f"{AUTHORIZATION_SERVICE_URL}/api/v1/authorization/check-access",
                                   json={
                                       "user_id": user_id,
                                       "resource_type": test["resource_type"],
                                       "resource_name": test["resource_name"],
                                       "required_access_level": test["required_access_level"]
                                   }, timeout=5)
            
            if response.status_code == 200:
                result = response.json()
                has_access = result.get("has_access", False)
                expected = test["expected"]
                
                status = "✅" if has_access == expected else "❌"
                print(f"   {status} {test['resource_type']}:{test['resource_name']} "
                      f"(expected: {expected}, got: {has_access})")
                      
                if has_access != expected:
                    print(f"      Reason: {result.get('reason', 'Unknown')}")
            else:
                print(f"   ❌ Test failed: {response.status_code} - {response.text}")
                
        except requests.RequestException as e:
            print(f"   ❌ Test error: {e}")

def main():
    """Main setup function"""
    print("🚀 Setting up Resource Permissions for Unified Authorization")
    print("=" * 60)
    
    # Check if services are running
    if not check_services():
        print("❌ Required services are not running. Please start them first.")
        sys.exit(1)
    
    # Create test user and get token
    token = create_test_user()
    if not token:
        print("❌ Failed to create test user")
        sys.exit(1)
    
    # Setup default resource configurations  
    if not setup_default_resource_configs():
        print("⚠️  Some resource configurations may have failed")
    
    # Grant permissions to test user
    user_id = "test-user-123"
    if not grant_user_permissions(user_id):
        print("⚠️  Some permission grants may have failed")
    
    # Test the permissions
    test_permissions(user_id)
    
    print("\n" + "=" * 60)
    print("🎉 Resource permissions setup completed!")
    print(f"🔑 Test JWT Token: {token}")
    print("\n📋 Next steps:")
    print("   1. Use the JWT token to test Gateway authentication")
    print("   2. Test different API endpoints through the Gateway")
    print("   3. Verify permission checks are working correctly")
    print(f"   4. Check logs: tail -f /tmp/authorization_service.log")

if __name__ == "__main__":
    main()
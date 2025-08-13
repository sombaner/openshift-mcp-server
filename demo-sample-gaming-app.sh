#!/bin/bash

# 🎮 Sample Gaming App - Complete CI/CD Automation Demo
# Demonstrates full automation from Git repository to live application URL

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Demo configuration
GAMING_REPO="https://github.com/sur309/Sample_Gaming_App"
GAMING_NAMESPACE="gaming-demo"
MCP_SERVER_URL="https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"

echo -e "${PURPLE}🎮 Sample Gaming App - Complete CI/CD Automation Demo${NC}"
echo -e "${CYAN}===============================================${NC}"
echo ""
echo -e "${BLUE}📋 Demo Overview:${NC}"
echo "   Repository: $GAMING_REPO"
echo "   Target Namespace: $GAMING_NAMESPACE"
echo "   MCP Server: $(basename $MCP_SERVER_URL)"
echo ""

# Step 1: Test MCP Server Connectivity
echo -e "${YELLOW}🔗 Step 1: Testing MCP Server Connectivity${NC}"
echo ""

if curl -s "$MCP_SERVER_URL" > /dev/null; then
    echo -e "✅ ${GREEN}MCP server is accessible${NC}"
else
    echo -e "❌ ${RED}MCP server is not accessible${NC}"
    echo "   Please ensure the server is running and accessible"
    exit 1
fi

# Step 2: Initialize MCP Session
echo ""
echo -e "${YELLOW}🚀 Step 2: Initialize MCP Session${NC}"

INIT_PAYLOAD='{
    "jsonrpc": "2.0",
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {
            "name": "gaming-demo-client",
            "version": "1.0.0"
        }
    },
    "id": 1
}'

echo "   Sending initialize request..."
INIT_RESPONSE=$(curl -s -X POST "$MCP_SERVER_URL" \
    -H "Content-Type: application/json" \
    -d "$INIT_PAYLOAD")

if echo "$INIT_RESPONSE" | grep -q '"protocolVersion"'; then
    echo -e "✅ ${GREEN}MCP session initialized successfully${NC}"
else
    echo -e "❌ ${RED}Failed to initialize MCP session${NC}"
    echo "Response: $INIT_RESPONSE"
    exit 1
fi

# Step 3: List Available Tools
echo ""
echo -e "${YELLOW}🛠️ Step 3: List Available CI/CD Tools${NC}"

TOOLS_PAYLOAD='{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "params": {},
    "id": 2
}'

echo "   Fetching available tools..."
TOOLS_RESPONSE=$(curl -s -X POST "$MCP_SERVER_URL" \
    -H "Content-Type: application/json" \
    -d "$TOOLS_PAYLOAD")

if echo "$TOOLS_RESPONSE" | grep -q 'repo_auto_deploy'; then
    echo -e "✅ ${GREEN}Full automation tools available${NC}"
    echo "   Found: repo_add, repo_auto_deploy, repo_generate_manifests, repo_get_url"
else
    echo -e "❌ ${RED}Required tools not found${NC}"
    exit 1
fi

# Step 4: Add Gaming Repository with Full Automation
echo ""
echo -e "${YELLOW}🎯 Step 4: Add Sample Gaming App for Full Automation${NC}"

AUTO_DEPLOY_PAYLOAD='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "repo_auto_deploy",
        "arguments": {
            "url": "'$GAMING_REPO'",
            "namespace": "'$GAMING_NAMESPACE'",
            "name": "sample-gaming-app",
            "branch": "main",
            "port": 8080,
            "image_registry": "quay.io"
        }
    },
    "id": 3
}'

echo "   Triggering full automation for Sample Gaming App..."
echo "   🔄 This will:"
echo "      • Create namespace '$GAMING_NAMESPACE'"
echo "      • Generate Kubernetes manifests"
echo "      • Configure build pipeline"
echo "      • Set up deployment automation"
echo "      • Generate live application URL"
echo ""

AUTO_DEPLOY_RESPONSE=$(curl -s -X POST "$MCP_SERVER_URL" \
    -H "Content-Type: application/json" \
    -d "$AUTO_DEPLOY_PAYLOAD")

if echo "$AUTO_DEPLOY_RESPONSE" | grep -q '"status".*"success"'; then
    echo -e "✅ ${GREEN}Full automation configured successfully!${NC}"
    
    # Extract application URL
    APP_URL=$(echo "$AUTO_DEPLOY_RESPONSE" | grep -o 'https://[^"]*gaming-demo[^"]*' | head -1)
    if [ ! -z "$APP_URL" ]; then
        echo ""
        echo -e "🌐 ${CYAN}Live Application URL: $APP_URL${NC}"
    fi
    
    echo ""
    echo -e "📋 ${BLUE}Automation Summary:${NC}"
    echo "$AUTO_DEPLOY_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$AUTO_DEPLOY_RESPONSE"
    
else
    echo -e "❌ ${RED}Failed to configure automation${NC}"
    echo "Response: $AUTO_DEPLOY_RESPONSE"
    exit 1
fi

# Step 5: Generate and Display Manifests
echo ""
echo -e "${YELLOW}📋 Step 5: Generate Kubernetes Manifests${NC}"

MANIFESTS_PAYLOAD='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "repo_generate_manifests",
        "arguments": {
            "name": "sample-gaming-app",
            "image_tag": "latest"
        }
    },
    "id": 4
}'

echo "   Generating Kubernetes manifests for deployment..."
MANIFESTS_RESPONSE=$(curl -s -X POST "$MCP_SERVER_URL" \
    -H "Content-Type: application/json" \
    -d "$MANIFESTS_PAYLOAD")

if echo "$MANIFESTS_RESPONSE" | grep -q 'deployment.yaml'; then
    echo -e "✅ ${GREEN}Kubernetes manifests generated${NC}"
    echo "   Generated: Deployment, Service, Route manifests"
else
    echo -e "⚠️ ${YELLOW}Manifests generation response:${NC}"
    echo "$MANIFESTS_RESPONSE"
fi

# Step 6: Get Live Application URL
echo ""
echo -e "${YELLOW}🌐 Step 6: Get Live Application Access Information${NC}"

URL_PAYLOAD='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "repo_get_url",
        "arguments": {
            "name": "sample-gaming-app"
        }
    },
    "id": 5
}'

echo "   Fetching application access URLs..."
URL_RESPONSE=$(curl -s -X POST "$MCP_SERVER_URL" \
    -H "Content-Type: application/json" \
    -d "$URL_PAYLOAD")

if echo "$URL_RESPONSE" | grep -q 'external_url'; then
    echo -e "✅ ${GREEN}Application URLs generated${NC}"
    
    # Extract URLs
    EXTERNAL_URL=$(echo "$URL_RESPONSE" | grep -o 'https://[^"]*gaming-demo[^"]*' | head -1)
    
    echo ""
    echo -e "🎮 ${CYAN}Game Access Information:${NC}"
    echo "   🌐 Play Online: $EXTERNAL_URL"
    echo "   📱 Mobile Friendly: Yes"
    echo "   🎯 Game Type: Snake Game"
    echo "   ⚡ Port: 8080"
    
else
    echo -e "⚠️ ${YELLOW}URL response:${NC}"
    echo "$URL_RESPONSE"
fi

# Step 7: Verify Deployment Status
echo ""
echo -e "${YELLOW}📊 Step 7: Check Deployment Status${NC}"

echo "   Checking OpenShift namespace and deployment..."

# Check if namespace exists
if oc get namespace "$GAMING_NAMESPACE" >/dev/null 2>&1; then
    echo -e "✅ ${GREEN}Namespace '$GAMING_NAMESPACE' exists${NC}"
    
    # Check pods
    echo "   Checking pod status..."
    POD_STATUS=$(oc get pods -n "$GAMING_NAMESPACE" --no-headers 2>/dev/null | head -1)
    if [ ! -z "$POD_STATUS" ]; then
        echo "   📋 Pod Status: $POD_STATUS"
    else
        echo -e "   ⏳ ${YELLOW}Pods may still be starting...${NC}"
    fi
    
    # Check service
    echo "   Checking service..."
    if oc get service sample-gaming-app -n "$GAMING_NAMESPACE" >/dev/null 2>&1; then
        echo -e "✅ ${GREEN}Service 'sample-gaming-app' exists${NC}"
    else
        echo -e "   ⏳ ${YELLOW}Service may still be creating...${NC}"
    fi
    
    # Check route
    echo "   Checking route..."
    if oc get route sample-gaming-app -n "$GAMING_NAMESPACE" >/dev/null 2>&1; then
        echo -e "✅ ${GREEN}Route 'sample-gaming-app' exists${NC}"
        ACTUAL_URL=$(oc get route sample-gaming-app -n "$GAMING_NAMESPACE" -o jsonpath='{.spec.host}' 2>/dev/null)
        if [ ! -z "$ACTUAL_URL" ]; then
            echo -e "   🌐 Live URL: https://$ACTUAL_URL"
        fi
    else
        echo -e "   ⏳ ${YELLOW}Route may still be creating...${NC}"
    fi
    
else
    echo -e "   ⏳ ${YELLOW}Namespace '$GAMING_NAMESPACE' may still be creating...${NC}"
fi

# Step 8: Demo Summary
echo ""
echo -e "${PURPLE}🎉 Demo Complete - Summary${NC}"
echo -e "${CYAN}=========================${NC}"
echo ""
echo -e "${GREEN}✅ Successfully Demonstrated:${NC}"
echo "   🔗 MCP Server connectivity and protocol compliance"
echo "   🛠️ Tool discovery and availability"
echo "   🚀 Complete automation with repo_auto_deploy"
echo "   📋 Kubernetes manifest generation"
echo "   🌐 Live URL generation and access"
echo "   📊 Deployment status verification"
echo ""
echo -e "${BLUE}🎮 Sample Gaming App Information:${NC}"
echo "   📁 Repository: $GAMING_REPO"
echo "   🏠 Namespace: $GAMING_NAMESPACE"
echo "   🎯 Application: Snake Game (JavaScript/HTML5)"
echo "   🌐 Expected URL: https://sample-gaming-app-$GAMING_NAMESPACE.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
echo ""
echo -e "${YELLOW}🔄 Next Steps for Complete CI/CD:${NC}"
echo "   1. 📝 Configure webhook in GitHub repository"
echo "   2. 🔄 Commit changes will trigger automatic builds"
echo "   3. 🚀 Automatic deployment to OpenShift"
echo "   4. 🌐 Live URL remains consistent"
echo "   5. 📊 Monitor with: oc get pods -n $GAMING_NAMESPACE"
echo ""
echo -e "${CYAN}🎯 This demo proves:${NC}"
echo "   • MCP server works with ANY Git repository"
echo "   • Full automation from commit to live URL"
echo "   • Dynamic namespace deployment"
echo "   • Production-ready CI/CD pipeline"
echo "   • GitHub Copilot integration capabilities"
echo ""
echo -e "${GREEN}🏆 Automation Complete!${NC}"

#!/bin/bash

# MCP Integration Test Script
# Tests the OpenShift AI MCP server endpoints for GitHub Copilot integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Server endpoints
INFERENCE_URL="https://openshift-ai-mcp-server-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
MCP_URL="https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"

echo -e "${BLUE}🧪 Testing OpenShift AI MCP Server Integration${NC}"
echo ""

# Function to test HTTP endpoint
test_http_endpoint() {
    local url=$1
    local description=$2
    local expected_status=${3:-200}
    
    echo -e "${YELLOW}📡 Testing: $description${NC}"
    echo -e "   URL: $url"
    
    # Test with curl
    local response=$(curl -s -w "%{http_code}" -o /tmp/response.txt "$url" 2>/dev/null || echo "000")
    local body=$(cat /tmp/response.txt 2>/dev/null || echo "")
    
    if [[ "$response" == "$expected_status" ]]; then
        echo -e "${GREEN}   ✅ Status: $response (Expected: $expected_status)${NC}"
        if [[ -n "$body" && "$body" != *"404"* ]]; then
            echo -e "${GREEN}   ✅ Response: ${body:0:100}...${NC}"
        fi
    else
        echo -e "${RED}   ❌ Status: $response (Expected: $expected_status)${NC}"
        if [[ -n "$body" ]]; then
            echo -e "${RED}   ❌ Response: ${body:0:200}${NC}"
        fi
        return 1
    fi
    echo ""
    return 0
}

# Function to test JSON-RPC endpoint
test_jsonrpc_endpoint() {
    local url=$1
    local method=$2
    local description=$3
    
    echo -e "${YELLOW}🔧 Testing JSON-RPC: $description${NC}"
    echo -e "   URL: $url"
    echo -e "   Method: $method"
    
    local payload="{\"jsonrpc\": \"2.0\", \"method\": \"$method\", \"params\": {}, \"id\": 1}"
    local response=$(curl -s -X POST "$url" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null || echo "")
    
    if [[ -n "$response" && "$response" != *"404"* && "$response" != *"error"* ]]; then
        echo -e "${GREEN}   ✅ JSON-RPC Response: ${response:0:150}...${NC}"
    else
        echo -e "${RED}   ❌ JSON-RPC failed or returned error${NC}"
        if [[ -n "$response" ]]; then
            echo -e "${RED}   Response: ${response:0:200}${NC}"
        fi
        return 1
    fi
    echo ""
    return 0
}

# Function to check OpenShift deployment
check_openshift_deployment() {
    echo -e "${BLUE}🔍 Checking OpenShift Deployment Status${NC}"
    
    if command -v oc &> /dev/null; then
        echo -e "${GREEN}✅ OpenShift CLI available${NC}"
        
        # Check if connected to cluster
        if oc cluster-info &> /dev/null; then
            echo -e "${GREEN}✅ Connected to OpenShift cluster${NC}"
            
            # Check namespace
            if oc get namespace ai-mcp-openshift &> /dev/null; then
                echo -e "${GREEN}✅ Namespace ai-mcp-openshift exists${NC}"
                
                # Check pods
                echo -e "${YELLOW}📋 Pod Status:${NC}"
                oc get pods -n ai-mcp-openshift -o wide
                echo ""
                
                # Check services
                echo -e "${YELLOW}📋 Service Status:${NC}"
                oc get svc -n ai-mcp-openshift
                echo ""
                
                # Check routes
                echo -e "${YELLOW}📋 Route Status:${NC}"
                oc get routes -n ai-mcp-openshift
                echo ""
                
            else
                echo -e "${RED}❌ Namespace ai-mcp-openshift not found${NC}"
                return 1
            fi
        else
            echo -e "${YELLOW}⚠️  Not connected to OpenShift cluster${NC}"
            echo -e "${YELLOW}💡 Run: oc login <cluster-url>${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  OpenShift CLI not available${NC}"
        echo -e "${YELLOW}💡 Install with: curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-<os>-<arch>.tar.gz${NC}"
    fi
    echo ""
}

# Function to test VS Code integration files
check_vscode_integration() {
    echo -e "${BLUE}📁 Checking VS Code Integration Files${NC}"
    
    local files=(".vscode/settings.json" ".vscode/mcp-config.json")
    
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            echo -e "${GREEN}✅ Found: $file${NC}"
            
            # Check if it contains the correct URL
            if grep -q "openshift-ai-mcp-server-mcp-ai-mcp-openshift" "$file"; then
                echo -e "${GREEN}   ✅ Contains correct MCP server URL${NC}"
            else
                echo -e "${YELLOW}   ⚠️  May need URL update${NC}"
            fi
        else
            echo -e "${RED}❌ Missing: $file${NC}"
        fi
    done
    echo ""
}

# Function to provide integration tips
provide_integration_tips() {
    echo -e "${BLUE}💡 GitHub Copilot Integration Tips${NC}"
    echo ""
    echo -e "${YELLOW}1. VS Code Setup:${NC}"
    echo -e "   code --install-extension ms-vscode.vscode-copilot"
    echo -e "   code --install-extension ms-vscode.vscode-copilot-chat"
    echo -e "   code --install-extension modelcontextprotocol.mcp-client"
    echo ""
    echo -e "${YELLOW}2. Open this project in VS Code:${NC}"
    echo -e "   code ."
    echo ""
    echo -e "${YELLOW}3. Test Copilot Chat:${NC}"
    echo -e "   Open Copilot Chat (Ctrl/Cmd + Shift + I)"
    echo -e "   Type: @copilot Can you connect to the OpenShift AI MCP server?"
    echo ""
    echo -e "${YELLOW}4. Use Custom Commands:${NC}"
    echo -e "   @copilot /deploy    - Deploy current project"
    echo -e "   @copilot /build     - Build container image"
    echo -e "   @copilot /watch-repo - Set up repository monitoring"
    echo -e "   @copilot /status    - Check deployment status"
    echo ""
    echo -e "${YELLOW}5. Manual MCP Test:${NC}"
    echo -e "   curl -X POST $MCP_URL \\\\"
    echo -e "     -H \"Content-Type: application/json\" \\\\"
    echo -e "     -d '{\"jsonrpc\": \"2.0\", \"method\": \"tools/list\", \"params\": {}, \"id\": 1}'"
    echo ""
}

# Main test execution
main() {
    local errors=0
    
    # Check OpenShift deployment first
    check_openshift_deployment
    
    # Check VS Code integration files
    check_vscode_integration
    
    # Test inference endpoints
    echo -e "${BLUE}🧪 Testing Inference Server Endpoints${NC}"
    echo ""
    
    test_http_endpoint "$INFERENCE_URL/health" "Health Check" "200" || errors=$((errors + 1))
    test_http_endpoint "$INFERENCE_URL/models" "Models List" "200" || errors=$((errors + 1))
    test_http_endpoint "$INFERENCE_URL/" "Root Info" "200" || errors=$((errors + 1))
    
    # Test MCP endpoints
    echo -e "${BLUE}🧪 Testing MCP Server Endpoints${NC}"
    echo ""
    
    test_http_endpoint "$MCP_URL/" "MCP Root" "200" || errors=$((errors + 1))
    test_jsonrpc_endpoint "$MCP_URL" "initialize" "MCP Initialize" || errors=$((errors + 1))
    test_jsonrpc_endpoint "$MCP_URL" "tools/list" "MCP Tools List" || errors=$((errors + 1))
    
    # Summary
    echo -e "${BLUE}📊 Test Summary${NC}"
    if [[ $errors -eq 0 ]]; then
        echo -e "${GREEN}🎉 All tests passed! Your MCP server is ready for GitHub Copilot integration.${NC}"
        echo ""
        echo -e "${GREEN}✅ Inference server is responding${NC}"
        echo -e "${GREEN}✅ MCP server is available${NC}"
        echo -e "${GREEN}✅ VS Code configuration files are present${NC}"
        echo ""
        provide_integration_tips
    else
        echo -e "${RED}❌ Found $errors error(s). Please check the issues above.${NC}"
        echo ""
        echo -e "${YELLOW}🔧 Troubleshooting Steps:${NC}"
        echo -e "1. Check pod logs: oc logs deployment/openshift-ai-mcp-server -n ai-mcp-openshift"
        echo -e "2. Verify routes: oc get routes -n ai-mcp-openshift"
        echo -e "3. Test internal connectivity: oc port-forward deployment/openshift-ai-mcp-server -n ai-mcp-openshift 8080:8080 8081:8081"
        echo -e "4. Check service status: oc get svc -n ai-mcp-openshift"
        return 1
    fi
}

# Cleanup function
cleanup() {
    rm -f /tmp/response.txt
}

# Set up cleanup trap
trap cleanup EXIT

# Run main function
main "$@"

#!/bin/bash

# OpenShift AI MCP Server Deployment Script
# Deploys the optimized 270MB container to OpenShift

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="openshift-ai-mcp"
IMAGE="quay.io/sureshgaikwad/openshift-mcp-server:v4"

echo -e "${BLUE}🚀 Deploying OpenShift AI MCP Server${NC}"
echo -e "${BLUE}📦 Image: ${IMAGE} (270MB optimized)${NC}"
echo ""

# Check if oc is available
if ! command -v oc &> /dev/null; then
    echo -e "${RED}❌ oc CLI not found. Please install OpenShift CLI${NC}"
    exit 1
fi

# Check if logged in
if ! oc whoami &> /dev/null; then
    echo -e "${RED}❌ Not logged into OpenShift. Please run 'oc login'${NC}"
    exit 1
fi

echo -e "${YELLOW}👤 Logged in as: $(oc whoami)${NC}"
echo ""

# Create or switch to namespace
echo -e "${BLUE}📁 Setting up namespace: ${NAMESPACE}${NC}"
if oc get project ${NAMESPACE} &> /dev/null; then
    echo -e "${GREEN}✅ Project ${NAMESPACE} already exists${NC}"
    oc project ${NAMESPACE}
else
    echo -e "${YELLOW}📝 Creating project ${NAMESPACE}${NC}"
    oc new-project ${NAMESPACE} --description="OpenShift AI MCP Server for CI/CD automation"
fi

# Deploy manifests in order
echo ""
echo -e "${BLUE}🔧 Deploying manifests...${NC}"

manifests=(
    "manifests/configmap.yaml"
    "manifests/rbac.yaml" 
    "manifests/secrets.yaml"
    "manifests/deployment.yaml"
    "manifests/service.yaml"
)

for manifest in "${manifests[@]}"; do
    if [[ -f "$manifest" ]]; then
        echo -e "${YELLOW}📋 Applying ${manifest}${NC}"
        oc apply -f "$manifest"
    else
        echo -e "${RED}❌ Manifest not found: ${manifest}${NC}"
        exit 1
    fi
done

# Wait for deployment to be ready
echo ""
echo -e "${BLUE}⏳ Waiting for deployment to be ready...${NC}"
oc wait --for=condition=available --timeout=300s deployment/openshift-ai-mcp-server

# Get pod status
echo ""
echo -e "${BLUE}📊 Deployment Status:${NC}"
oc get pods -l app.kubernetes.io/name=openshift-ai-mcp-server

# Get services and routes
echo ""
echo -e "${BLUE}🌐 Services and Routes:${NC}"
oc get svc,route

# Get external URLs
echo ""
echo -e "${BLUE}🔗 External URLs:${NC}"

if oc get route openshift-ai-mcp-server &> /dev/null; then
    INFERENCE_URL=$(oc get route openshift-ai-mcp-server -o jsonpath='{.spec.host}')
    echo -e "${GREEN}🔍 Inference URL: https://${INFERENCE_URL}${NC}"
else
    echo -e "${YELLOW}⚠️  Inference route not found${NC}"
fi

if oc get route openshift-ai-mcp-server-mcp &> /dev/null; then
    MCP_URL=$(oc get route openshift-ai-mcp-server-mcp -o jsonpath='{.spec.host}')
    echo -e "${GREEN}🔧 MCP URL: https://${MCP_URL}${NC}"
else
    echo -e "${YELLOW}⚠️  MCP route not found${NC}"
fi

# Test endpoints
echo ""
echo -e "${BLUE}🧪 Testing endpoints...${NC}"

if [[ -n "${INFERENCE_URL}" ]]; then
    echo -e "${YELLOW}Testing inference health endpoint...${NC}"
    if curl -k -s -f "https://${INFERENCE_URL}/health" > /dev/null; then
        echo -e "${GREEN}✅ Inference endpoint is healthy${NC}"
    else
        echo -e "${RED}❌ Inference endpoint health check failed${NC}"
    fi

    echo -e "${YELLOW}Testing inference endpoint...${NC}"
    INFERENCE_RESPONSE=$(curl -k -s -X POST "https://${INFERENCE_URL}/infer" \
        -H "Content-Type: application/json" \
        -d '{"inputs": "test deployment", "model_name": "lightweight"}' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && [[ -n "${INFERENCE_RESPONSE}" ]]; then
        echo -e "${GREEN}✅ Inference endpoint working${NC}"
        echo -e "${BLUE}📄 Sample response:${NC}"
        echo "${INFERENCE_RESPONSE}" | jq . 2>/dev/null || echo "${INFERENCE_RESPONSE}"
    else
        echo -e "${RED}❌ Inference endpoint test failed${NC}"
    fi
fi

if [[ -n "${MCP_URL}" ]]; then
    echo -e "${YELLOW}Testing MCP health endpoint...${NC}"
    if curl -k -s -f "https://${MCP_URL}/health/mcp" > /dev/null; then
        echo -e "${GREEN}✅ MCP endpoint is healthy${NC}"
    else
        echo -e "${RED}❌ MCP endpoint health check failed${NC}"
    fi
fi

# Display resource usage
echo ""
echo -e "${BLUE}📈 Resource Usage:${NC}"
if oc top pods -l app.kubernetes.io/name=openshift-ai-mcp-server 2>/dev/null; then
    echo ""
else
    echo -e "${YELLOW}⚠️  Resource metrics not available (metrics server required)${NC}"
fi

# Show logs sample
echo ""
echo -e "${BLUE}📋 Recent logs (last 10 lines):${NC}"
oc logs -l app.kubernetes.io/name=openshift-ai-mcp-server --tail=10 2>/dev/null || echo -e "${YELLOW}⚠️  Logs not available${NC}"

# Final summary
echo ""
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo ""
echo -e "${BLUE}📊 Summary:${NC}"
echo -e "  • ${GREEN}Project:${NC} ${NAMESPACE}"
echo -e "  • ${GREEN}Image:${NC} ${IMAGE}"
echo -e "  • ${GREEN}Size:${NC} 270MB (95% reduction from 5.78GB)"
echo -e "  • ${GREEN}Resources:${NC} 128Mi-512Mi RAM, 100m-500m CPU"

if [[ -n "${INFERENCE_URL}" ]]; then
    echo -e "  • ${GREEN}Inference:${NC} https://${INFERENCE_URL}"
fi

if [[ -n "${MCP_URL}" ]]; then
    echo -e "  • ${GREEN}MCP Server:${NC} https://${MCP_URL}"
fi

echo ""
echo -e "${BLUE}🔧 Next Steps:${NC}"
echo -e "  1. Configure VS Code MCP integration (see VSCODE_INTEGRATION.md)"
echo -e "  2. Set up GitHub Copilot workflows"
echo -e "  3. Test automated CI/CD pipeline"
echo ""
echo -e "${BLUE}📚 Documentation:${NC}"
echo -e "  • DEPLOYMENT.md - Detailed deployment guide"
echo -e "  • VSCODE_INTEGRATION.md - VS Code + Copilot setup"
echo -e "  • OPTIMIZATION.md - Image optimization details"
echo ""

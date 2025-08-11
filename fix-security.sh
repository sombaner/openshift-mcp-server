#!/bin/bash

# Fix PodSecurity violations script
# Applies security context updates to existing deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

NAMESPACE="ai-mcp-openshift"
DEPLOYMENT="openshift-ai-mcp-server"

echo -e "${BLUE}🔒 Fixing PodSecurity violations for OpenShift AI MCP Server${NC}"
echo -e "${BLUE}📋 Namespace: ${NAMESPACE}${NC}"
echo -e "${BLUE}🚀 Deployment: ${DEPLOYMENT}${NC}"
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

# Switch to correct namespace
echo -e "${BLUE}📁 Switching to namespace: ${NAMESPACE}${NC}"
oc project ${NAMESPACE}

# Check if deployment exists
if ! oc get deployment ${DEPLOYMENT} &> /dev/null; then
    echo -e "${RED}❌ Deployment ${DEPLOYMENT} not found in namespace ${NAMESPACE}${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Found deployment ${DEPLOYMENT}${NC}"

# Apply the updated deployment with security fixes
echo -e "${BLUE}🔧 Applying security context fixes...${NC}"
oc apply -f manifests/deployment.yaml

# Wait for rollout to complete
echo -e "${BLUE}⏳ Waiting for deployment rollout to complete...${NC}"
oc rollout status deployment/${DEPLOYMENT} --timeout=300s

# Check pod status
echo ""
echo -e "${BLUE}📊 Pod Status:${NC}"
oc get pods -l app.kubernetes.io/name=${DEPLOYMENT}

# Verify security context
echo ""
echo -e "${BLUE}🔒 Verifying security context...${NC}"
POD_NAME=$(oc get pods -l app.kubernetes.io/name=${DEPLOYMENT} -o jsonpath='{.items[0].metadata.name}')

if [[ -n "${POD_NAME}" ]]; then
    echo -e "${YELLOW}📋 Checking pod security context for: ${POD_NAME}${NC}"
    
    # Check pod-level security context
    echo -e "${BLUE}Pod SecurityContext:${NC}"
    oc get pod ${POD_NAME} -o jsonpath='{.spec.securityContext}' | jq .
    
    echo ""
    echo -e "${BLUE}Container SecurityContext:${NC}"
    oc get pod ${POD_NAME} -o jsonpath='{.spec.containers[0].securityContext}' | jq .
    
    echo ""
    echo -e "${BLUE}🧪 Testing pod compliance...${NC}"
    
    # Check for specific security settings
    ALLOW_PRIV_ESC=$(oc get pod ${POD_NAME} -o jsonpath='{.spec.containers[0].securityContext.allowPrivilegeEscalation}')
    CAPS_DROP=$(oc get pod ${POD_NAME} -o jsonpath='{.spec.containers[0].securityContext.capabilities.drop[0]}')
    SECCOMP=$(oc get pod ${POD_NAME} -o jsonpath='{.spec.securityContext.seccompProfile.type}')
    RUN_AS_NON_ROOT=$(oc get pod ${POD_NAME} -o jsonpath='{.spec.securityContext.runAsNonRoot}')
    
    echo -e "${YELLOW}Security Settings Check:${NC}"
    
    if [[ "${ALLOW_PRIV_ESC}" == "false" ]]; then
        echo -e "${GREEN}✅ allowPrivilegeEscalation: false${NC}"
    else
        echo -e "${RED}❌ allowPrivilegeEscalation: ${ALLOW_PRIV_ESC}${NC}"
    fi
    
    if [[ "${CAPS_DROP}" == "ALL" ]]; then
        echo -e "${GREEN}✅ capabilities.drop: [ALL]${NC}"
    else
        echo -e "${RED}❌ capabilities.drop: ${CAPS_DROP}${NC}"
    fi
    
    if [[ "${SECCOMP}" == "RuntimeDefault" ]]; then
        echo -e "${GREEN}✅ seccompProfile.type: RuntimeDefault${NC}"
    else
        echo -e "${RED}❌ seccompProfile.type: ${SECCOMP}${NC}"
    fi
    
    if [[ "${RUN_AS_NON_ROOT}" == "true" ]]; then
        echo -e "${GREEN}✅ runAsNonRoot: true${NC}"
    else
        echo -e "${RED}❌ runAsNonRoot: ${RUN_AS_NON_ROOT}${NC}"
    fi
    
else
    echo -e "${RED}❌ No running pods found${NC}"
fi

# Show recent logs
echo ""
echo -e "${BLUE}📋 Recent logs (last 5 lines):${NC}"
oc logs -l app.kubernetes.io/name=${DEPLOYMENT} --tail=5 2>/dev/null || echo -e "${YELLOW}⚠️  Logs not available${NC}"

echo ""
echo -e "${GREEN}🎉 Security context fixes applied!${NC}"
echo ""
echo -e "${BLUE}📊 Summary:${NC}"
echo -e "  • ${GREEN}PodSecurity Standard:${NC} restricted:latest"
echo -e "  • ${GREEN}allowPrivilegeEscalation:${NC} false"
echo -e "  • ${GREEN}capabilities:${NC} drop=[ALL]"
echo -e "  • ${GREEN}seccompProfile:${NC} RuntimeDefault"
echo -e "  • ${GREEN}runAsNonRoot:${NC} true"
echo ""
echo -e "${BLUE}🔗 The deployment should now be compliant with OpenShift restricted security policies.${NC}"

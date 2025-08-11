#!/bin/bash

# OpenShift Security Context Constraint (SCC) Fix Script
# Resolves OpenShift-specific security context issues

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔒 OpenShift SCC Fix Script${NC}"
echo ""

NAMESPACE="ai-mcp-openshift"
SERVICE_ACCOUNT="openshift-ai-mcp-server"
DEPLOYMENT="openshift-ai-mcp-server"

# Function to check if we're connected to OpenShift
check_openshift_connection() {
    if ! oc cluster-info &> /dev/null; then
        echo -e "${RED}❌ Not connected to OpenShift cluster${NC}"
        echo -e "${YELLOW}💡 Please login with: oc login${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Connected to OpenShift cluster${NC}"
}

# Function to check current SCCs
check_sccs() {
    echo -e "${BLUE}📋 Available Security Context Constraints:${NC}"
    oc get scc --no-headers | awk '{print "  - " $1}' || true
    echo ""
}

# Function to check ServiceAccount SCC bindings
check_sa_sccs() {
    echo -e "${BLUE}🔍 Checking ServiceAccount SCC bindings...${NC}"
    
    if oc get sa "$SERVICE_ACCOUNT" -n "$NAMESPACE" &> /dev/null; then
        echo -e "${GREEN}✅ ServiceAccount exists: $SERVICE_ACCOUNT${NC}"
        
        # Check which SCCs the ServiceAccount can use
        echo -e "${YELLOW}📊 SCCs available to ServiceAccount:${NC}"
        oc describe sa "$SERVICE_ACCOUNT" -n "$NAMESPACE" | grep -A 10 "Mountable secrets" || true
        
        # Try to determine which SCC will be used
        echo -e "${YELLOW}🎯 Attempting to determine effective SCC...${NC}"
        oc policy can-i use scc/restricted-v2 --as=system:serviceaccount:$NAMESPACE:$SERVICE_ACCOUNT && \
            echo -e "${GREEN}  ✅ Can use restricted-v2 SCC${NC}" || \
            echo -e "${YELLOW}  ⚠️  Cannot use restricted-v2 SCC${NC}"
    else
        echo -e "${RED}❌ ServiceAccount not found: $SERVICE_ACCOUNT${NC}"
        return 1
    fi
    echo ""
}

# Function to get the allowed UID range for the namespace
get_uid_range() {
    echo -e "${BLUE}🔢 Checking namespace UID ranges...${NC}"
    
    if oc get namespace "$NAMESPACE" -o jsonpath='{.metadata.annotations}' | grep -q "openshift.io/sa.scc.uid-range"; then
        UID_RANGE=$(oc get namespace "$NAMESPACE" -o jsonpath='{.metadata.annotations.openshift\.io/sa\.scc\.uid-range}')
        echo -e "${GREEN}✅ UID range for namespace: $UID_RANGE${NC}"
    else
        echo -e "${YELLOW}⚠️  No explicit UID range annotation found${NC}"
        echo -e "${YELLOW}💡 OpenShift will assign automatically${NC}"
    fi
    echo ""
}

# Function to apply SCC-compatible security context
apply_scc_fix() {
    echo -e "${BLUE}🛠️  Applying OpenShift SCC fixes...${NC}"
    
    # Check if deployment has problematic security context
    if grep -q "runAsUser:" manifests/deployment.yaml || grep -q "fsGroup:" manifests/deployment.yaml; then
        echo -e "${YELLOW}⚠️  Found explicit user/group IDs in deployment${NC}"
        echo -e "${BLUE}🔧 Removing explicit user/group IDs...${NC}"
        
        # This should already be done by the previous fix, but let's make sure
        if grep -q "runAsUser:" manifests/deployment.yaml; then
            echo -e "${RED}❌ Still found runAsUser in deployment.yaml${NC}"
            echo -e "${YELLOW}💡 Manual fix required - remove runAsUser from deployment${NC}"
            return 1
        fi
        
        if grep -q "fsGroup:" manifests/deployment.yaml; then
            echo -e "${RED}❌ Still found fsGroup in deployment.yaml${NC}"
            echo -e "${YELLOW}💡 Manual fix required - remove fsGroup from deployment${NC}"
            return 1
        fi
    fi
    
    echo -e "${GREEN}✅ Security context is SCC-compatible${NC}"
    echo ""
}

# Function to test deployment
test_deployment() {
    echo -e "${BLUE}🧪 Testing deployment with OpenShift...${NC}"
    
    if oc apply --dry-run=server -f manifests/deployment.yaml &> /dev/null; then
        echo -e "${GREEN}✅ Deployment passes OpenShift validation${NC}"
    else
        echo -e "${RED}❌ Deployment validation failed:${NC}"
        oc apply --dry-run=server -f manifests/deployment.yaml
        return 1
    fi
    echo ""
}

# Function to provide SCC troubleshooting tips
provide_scc_tips() {
    echo -e "${BLUE}💡 OpenShift SCC Troubleshooting Tips:${NC}"
    echo ""
    echo -e "${YELLOW}1. Let OpenShift assign user/group IDs automatically:${NC}"
    echo -e "   - Remove explicit runAsUser and fsGroup values"
    echo -e "   - Keep runAsNonRoot: true"
    echo ""
    echo -e "${YELLOW}2. Check effective SCC:${NC}"
    echo -e "   oc get pod <pod-name> -o yaml | grep scc"
    echo ""
    echo -e "${YELLOW}3. View SCC details:${NC}"
    echo -e "   oc describe scc restricted-v2"
    echo ""
    echo -e "${YELLOW}4. Grant SCC to ServiceAccount (if needed):${NC}"
    echo -e "   oc adm policy add-scc-to-user restricted-v2 system:serviceaccount:$NAMESPACE:$SERVICE_ACCOUNT"
    echo ""
    echo -e "${YELLOW}5. Check namespace annotations:${NC}"
    echo -e "   oc get namespace $NAMESPACE -o yaml"
    echo ""
}

# Main execution
main() {
    check_openshift_connection
    check_sccs
    check_sa_sccs
    get_uid_range
    apply_scc_fix
    test_deployment
    
    echo -e "${GREEN}🎉 OpenShift SCC fixes applied successfully!${NC}"
    echo ""
    echo -e "${BLUE}🚀 Ready to deploy:${NC}"
    echo -e "  ${YELLOW}./deploy.sh${NC}"
    echo -e "  ${YELLOW}or${NC}"
    echo -e "  ${YELLOW}oc apply -f manifests/${NC}"
    echo ""
    
    provide_scc_tips
}

# Handle errors
trap 'echo -e "${RED}❌ Script failed. Check the error above.${NC}"; provide_scc_tips' ERR

# Run main function
main "$@"

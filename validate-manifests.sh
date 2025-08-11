#!/bin/bash

# Validate Kubernetes manifests script
# Checks syntax and validates manifests before deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Validating Kubernetes Manifests${NC}"
echo ""

# Check if oc or kubectl is available
if command -v oc &> /dev/null; then
    CLI="oc"
    echo -e "${GREEN}✅ Using OpenShift CLI (oc)${NC}"
elif command -v kubectl &> /dev/null; then
    CLI="kubectl"
    echo -e "${GREEN}✅ Using Kubernetes CLI (kubectl)${NC}"
else
    echo -e "${RED}❌ Neither oc nor kubectl found. Please install one of them.${NC}"
    exit 1
fi

# Function to validate a manifest file
validate_manifest() {
    local file=$1
    local filename=$(basename "$file")
    
    echo -e "${YELLOW}📋 Validating: ${filename}${NC}"
    
    if [[ ! -f "$file" ]]; then
        echo -e "${RED}❌ File not found: $file${NC}"
        return 1
    fi
    
    # Check YAML syntax
    if ! ${CLI} apply --dry-run=client -f "$file" > /dev/null 2>&1; then
        echo -e "${RED}❌ YAML syntax error in: $filename${NC}"
        ${CLI} apply --dry-run=client -f "$file"
        return 1
    fi
    
    # Validate with server (if connected)
    if ${CLI} cluster-info &> /dev/null; then
        if ! ${CLI} apply --dry-run=server -f "$file" > /dev/null 2>&1; then
            echo -e "${RED}❌ Server validation failed for: $filename${NC}"
            ${CLI} apply --dry-run=server -f "$file"
            return 1
        else
            echo -e "${GREEN}✅ Server validation passed: $filename${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Not connected to cluster - skipping server validation${NC}"
        echo -e "${GREEN}✅ Client validation passed: $filename${NC}"
    fi
    
    return 0
}

# Validate all manifest files
MANIFESTS=(
    "manifests/namespace.yaml"
    "manifests/configmap.yaml"
    "manifests/rbac.yaml"
    "manifests/secrets.yaml"
    "manifests/deployment.yaml"
    "manifests/service.yaml"
)

ERRORS=0

echo -e "${BLUE}🧪 Validating individual manifests...${NC}"
echo ""

for manifest in "${MANIFESTS[@]}"; do
    if ! validate_manifest "$manifest"; then
        ERRORS=$((ERRORS + 1))
    fi
    echo ""
done

# Validate complete manifest set
echo -e "${BLUE}🔧 Validating complete manifest set...${NC}"
if ${CLI} apply --dry-run=client -f manifests/ > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Complete manifest set validation passed${NC}"
else
    echo -e "${RED}❌ Complete manifest set validation failed${NC}"
    ${CLI} apply --dry-run=client -f manifests/
    ERRORS=$((ERRORS + 1))
fi

echo ""

# Check specific deployment requirements
echo -e "${BLUE}🔍 Checking Deployment specific requirements...${NC}"

if [[ -f "manifests/deployment.yaml" ]]; then
    # Check for required fields
    if grep -q "selector:" manifests/deployment.yaml; then
        echo -e "${GREEN}✅ Deployment has selector${NC}"
    else
        echo -e "${RED}❌ Deployment missing selector${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    
    if grep -A 5 "template:" manifests/deployment.yaml | grep -q "labels:"; then
        echo -e "${GREEN}✅ Deployment has template labels${NC}"
    else
        echo -e "${RED}❌ Deployment missing template labels${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    
    # Extract and compare selector and template labels
    echo -e "${YELLOW}📋 Checking selector/label consistency...${NC}"
    
    # This is a basic check - in production you'd want more sophisticated validation
    if grep -A 5 "selector:" manifests/deployment.yaml | grep -q "app.kubernetes.io/name" && \
       grep -A 5 "template:" manifests/deployment.yaml | grep -A 10 "labels:" | grep -q "app.kubernetes.io/name"; then
        echo -e "${GREEN}✅ Selector and template labels appear consistent${NC}"
    else
        echo -e "${RED}❌ Selector and template labels may not match${NC}"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""

# Security context validation
echo -e "${BLUE}🔒 Checking Security Context compliance...${NC}"

if [[ -f "manifests/deployment.yaml" ]]; then
    if grep -q "allowPrivilegeEscalation: false" manifests/deployment.yaml; then
        echo -e "${GREEN}✅ allowPrivilegeEscalation set to false${NC}"
    else
        echo -e "${RED}❌ allowPrivilegeEscalation not set to false${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    
    if grep -q "drop:" manifests/deployment.yaml && grep -A 2 "drop:" manifests/deployment.yaml | grep -q "ALL"; then
        echo -e "${GREEN}✅ Capabilities dropped to ALL${NC}"
    else
        echo -e "${RED}❌ Capabilities not properly dropped${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    
    if grep -q "seccompProfile" manifests/deployment.yaml; then
        echo -e "${GREEN}✅ seccompProfile configured${NC}"
    else
        echo -e "${RED}❌ seccompProfile not configured${NC}"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""

# Summary
if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}🎉 All validations passed! Manifests are ready for deployment.${NC}"
    echo ""
    echo -e "${BLUE}🚀 You can now safely deploy with:${NC}"
    echo -e "  ${YELLOW}./deploy.sh${NC}"
    echo -e "  ${YELLOW}or${NC}"
    echo -e "  ${YELLOW}oc apply -f manifests/${NC}"
else
    echo -e "${RED}❌ Found $ERRORS validation error(s). Please fix before deploying.${NC}"
    exit 1
fi

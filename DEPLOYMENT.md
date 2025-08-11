# OpenShift AI MCP Server Deployment Guide

## Overview
Deploy the optimized 270MB OpenShift AI MCP Server for CI/CD automation and VS Code integration with GitHub Copilot.

## Prerequisites

### OpenShift Cluster
- OpenShift 4.12+ or OpenShift AI cluster
- Cluster admin access or namespace admin permissions
- `oc` CLI tool installed and configured

### Container Registry
- Image pushed to: `quay.io/sureshgaikwad/openshift-mcp-server:v4`
- Size: 270MB (95% smaller than original)

## Deployment Steps

### 1. Create Project/Namespace
```bash
oc new-project ai-mcp-openshift
# or
oc create namespace ai-mcp-openshift
oc project ai-mcp-openshift
```

### 2. Apply Manifests
```bash
# Deploy in order
oc apply -f manifests/namespace.yaml
oc apply -f manifests/configmap.yaml
oc apply -f manifests/rbac.yaml
oc apply -f manifests/secrets.yaml
oc apply -f manifests/deployment.yaml
oc apply -f manifests/service.yaml
```

### 3. Verify Deployment
```bash
# Check pod status
oc get pods -l app.kubernetes.io/name=ai-mcp-openshift-server

# Check services
oc get svc ai-mcp-openshift-server

# Check routes
oc get route

# View logs
oc logs -l app.kubernetes.io/name=ai-mcp-openshift-server -f
```

### 4. Get External URLs
```bash
# Inference endpoint
INFERENCE_URL=$(oc get route ai-mcp-openshift-server -o jsonpath='{.spec.host}')
echo "Inference URL: https://$INFERENCE_URL"

# MCP endpoint
MCP_URL=$(oc get route ai-mcp-openshift-server-mcp -o jsonpath='{.spec.host}')
echo "MCP URL: https://$MCP_URL"
```

### 5. Test Deployment
```bash
# Test inference endpoint
curl -k https://$INFERENCE_URL/health

# Test inference
curl -k -X POST https://$INFERENCE_URL/infer \
  -H "Content-Type: application/json" \
  -d '{"inputs": "test", "model_name": "lightweight"}'

# Test MCP endpoint
curl -k https://$MCP_URL/health/mcp
```

## Resource Configuration

### Optimized Resources (270MB Image)
```yaml
resources:
  requests:
    memory: "128Mi"  # Was 512Mi
    cpu: "100m"      # Was 250m
  limits:
    memory: "512Mi"  # Was 2Gi
    cpu: "500m"      # Was 1000m
```

### Performance Benefits
- **Startup Time**: < 5 seconds (was 60+ seconds)
- **Memory Usage**: ~100MB runtime (was 1GB+)
- **Pull Time**: < 30 seconds (was 5+ minutes)
- **Storage**: 270MB (was 5.78GB)

## Troubleshooting

### Common Issues

#### Pod CrashLoopBackOff
```bash
# Check logs
oc logs -l app.kubernetes.io/name=ai-mcp-openshift-server --previous

# Check events
oc get events --sort-by=.metadata.creationTimestamp
```

#### ImagePullBackOff
```bash
# Verify image exists
podman pull quay.io/sureshgaikwad/openshift-mcp-server:v4

# Check pull secrets if using private registry
oc get secrets
```

#### Service Unavailable
```bash
# Check service endpoints
oc get endpoints ai-mcp-openshift-server

# Verify pod labels match service selector
oc get pods --show-labels
```

### Health Checks
```bash
# Pod health
oc exec -it deployment/ai-mcp-openshift-server -- curl localhost:8080/health

# Service health
oc exec -it deployment/ai-mcp-openshift-server -- curl localhost:8081/health/mcp
```

## Scaling

### Horizontal Scaling
```bash
# Scale up for high availability
oc scale deployment ai-mcp-openshift-server --replicas=3

# Auto-scaling (if HPA is available)
oc autoscale deployment ai-mcp-openshift-server --min=1 --max=5 --cpu-percent=70
```

### Resource Tuning
```bash
# Monitor resource usage
oc top pods -l app.kubernetes.io/name=ai-mcp-openshift-server

# Adjust resources if needed
oc patch deployment ai-mcp-openshift-server -p '{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "inference-server",
          "resources": {
            "requests": {"memory": "256Mi", "cpu": "200m"},
            "limits": {"memory": "1Gi", "cpu": "1000m"}
          }
        }]
      }
    }
  }
}'
```

## Security

### RBAC Permissions
The deployment includes minimal RBAC permissions for CI/CD operations:
- Pod creation/deletion
- Deployment management
- Service account token access
- Config map and secret read access

### Network Policies
```bash
# Apply network policy for additional security
oc apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ai-mcp-openshift-server-netpol
  namespace: ai-mcp-openshift
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: ai-mcp-openshift-server
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8081
  egress:
  - {}
EOF
```

## Monitoring

### Prometheus Metrics
The deployment is configured for Prometheus scraping:
```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

### Custom Metrics Endpoint
```bash
curl -k https://$INFERENCE_URL/metrics
```

## Cleanup

### Remove Deployment
```bash
# Delete all resources
oc delete -f manifests/

# Delete project (if dedicated)
oc delete project ai-mcp-openshift
```

### Verify Cleanup
```bash
# Check no resources remain
oc get all -n ai-mcp-openshift
```

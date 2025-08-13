# OpenShift AI MCP Server with CI/CD Automation

A comprehensive Model Context Protocol (MCP) server that runs on OpenShift AI, providing both ML inference capabilities and automated CI/CD pipelines for Kubernetes/OpenShift deployments.

## 🚀 Features

### ML Inference
- **Multi-Model Support**: PyTorch, transformers, scikit-learn
- **Text & Numeric Processing**: Handle both text embeddings and numeric data
- **Auto-Scaling**: Horizontal pod autoscaling based on load
- **Model Registry**: Dynamic model loading and management

### CI/CD Automation
- **Git Repository Monitoring**: Automatic detection of new commits
- **Container Image Building**: Docker and OpenShift BuildConfigs
- **Registry Management**: Push to multiple container registries
- **Automated Deployment**: Zero-downtime deployments to OpenShift
- **Pipeline Management**: End-to-end CI/CD pipeline orchestration

### MCP Integration
- **Native Kubernetes API**: Direct interaction without external tools
- **OpenShift Extensions**: Routes, BuildConfigs, ImageStreams
- **Real-time Monitoring**: Live status updates and event streaming
- **Security**: RBAC integration and secure credential management

## 📋 Prerequisites

- OpenShift 4.10+ or Kubernetes 1.21+
- Container registry access (Quay.io, Docker Hub, etc.)
- Git repository access
- OpenShift AI or similar ML platform (optional but recommended)

## 🛠️ Installation

### Quick Start: Deploy to OpenShift

```bash
# Clone the repository
git clone https://github.com/sur309/openshift-mcp-server.git
cd openshift-mcp-server

# Create namespace and apply manifests
oc apply -f manifests/namespace.yaml
oc apply -f manifests/rbac.yaml
oc apply -f manifests/configmap.yaml
oc apply -f manifests/secrets.yaml
oc apply -f manifests/deployment.yaml
oc apply -f manifests/service.yaml

# Apply Security Context Constraints (SCC) for the service account
oc adm policy add-scc-to-user anyuid -z openshift-ai-mcp-server -n ai-mcp-openshift
```

## 🚀 OpenShift Deployment Guide

### Prerequisites for OpenShift

- OpenShift 4.10+ cluster access
- `oc` CLI tool installed and authenticated
- Cluster admin privileges for SCC management
- Container registry credentials (Quay.io, Docker Hub, etc.)

### Step-by-Step OpenShift Deployment

#### 1. **Clone and Navigate to Repository**

```bash
git clone https://github.com/sur309/openshift-mcp-server.git
cd openshift-mcp-server
```

#### 2. **Create Project/Namespace**

```bash
# Create the project namespace
oc apply -f manifests/namespace.yaml

# Alternatively, create using oc new-project
# oc new-project ai-mcp-openshift --display-name="AI MCP OpenShift Server"
```

#### 3. **Apply Security Context Constraints (Critical for OpenShift)**

OpenShift requires specific Security Context Constraints (SCC) to allow the MCP server to run with the necessary permissions:

```bash
# Add anyuid SCC to the service account
oc adm policy add-scc-to-user anyuid -z openshift-ai-mcp-server -n ai-mcp-openshift

# Verify the SCC assignment
oc describe scc anyuid | grep -A 10 "Users:"
oc get scc anyuid -o yaml | grep -A 20 users:
```

#### 4. **Configure RBAC (Role-Based Access Control)**

```bash
# Apply service account and cluster-wide permissions
oc apply -f manifests/rbac.yaml

# Verify RBAC configuration
oc get serviceaccount openshift-ai-mcp-server -n ai-mcp-openshift
oc get clusterrolebinding openshift-ai-mcp-server
```

#### 5. **Create Configuration and Secrets**

```bash
# Apply configuration
oc apply -f manifests/configmap.yaml

# Create registry credentials secret
oc create secret generic registry-credentials \
  --from-literal=username=YOUR_REGISTRY_USERNAME \
  --from-literal=password=YOUR_REGISTRY_TOKEN \
  --from-literal=email=YOUR_EMAIL \
  -n ai-mcp-openshift

# Create Git credentials secret
oc create secret generic git-credentials \
  --from-literal=username=YOUR_GIT_USERNAME \
  --from-literal=token=YOUR_GIT_TOKEN \
  -n ai-mcp-openshift

# Create webhook secret for CI/CD
oc create secret generic webhook-secret \
  --from-literal=secret=$(openssl rand -hex 32) \
  -n ai-mcp-openshift

# Apply the secrets manifest (if using file-based secrets)
# oc apply -f manifests/secrets.yaml
```

#### 6. **Deploy the Application**

```bash
# Deploy the MCP server
oc apply -f manifests/deployment.yaml

# Create the service
oc apply -f manifests/service.yaml

# Verify deployment status
oc get pods -n ai-mcp-openshift
oc get deployment openshift-ai-mcp-server -n ai-mcp-openshift
```

#### 7. **Create OpenShift Route (for external access)**

```bash
# Create route for external access
oc expose service openshift-ai-mcp-server --port=8080 -n ai-mcp-openshift

# Create route for MCP endpoint
oc create route edge openshift-ai-mcp-server-mcp \
  --service=openshift-ai-mcp-server \
  --port=8081 \
  -n ai-mcp-openshift

# Get route URLs
oc get routes -n ai-mcp-openshift
```

#### 8. **Verify Deployment**

```bash
# Check pod status
oc get pods -n ai-mcp-openshift -o wide

# View application logs
oc logs -f deployment/openshift-ai-mcp-server -n ai-mcp-openshift

# Test health endpoints
ROUTE_URL=$(oc get route openshift-ai-mcp-server -o jsonpath='{.spec.host}' -n ai-mcp-openshift)
curl -k https://$ROUTE_URL/health
curl -k https://$ROUTE_URL:8081/health/mcp
```

### OpenShift-Specific Features

#### **Security Context Constraints (SCC) Explained**

The MCP server requires the `anyuid` SCC because it:
- Needs to run container builds and manage container registries
- Requires access to Docker/Podman socket for CI/CD operations  
- May need to run with specific user IDs for compatibility

```bash
# Check current SCC assignments
oc get scc anyuid -o yaml

# Verify service account has proper SCC
oc describe serviceaccount openshift-ai-mcp-server -n ai-mcp-openshift
```

#### **OpenShift Routes vs Services**

```bash
# Internal access (within cluster)
oc get svc openshift-ai-mcp-server -n ai-mcp-openshift

# External access (internet-facing)
oc get routes -n ai-mcp-openshift

# Test internal service
oc port-forward svc/openshift-ai-mcp-server 8080:8080 -n ai-mcp-openshift
```

### Troubleshooting OpenShift Deployment

#### **Common OpenShift Issues**

1. **SCC Permission Denied**
   ```bash
   # Error: pods "openshift-ai-mcp-server-xxx" is forbidden: unable to validate against any security context constraint
   oc adm policy add-scc-to-user anyuid -z openshift-ai-mcp-server -n ai-mcp-openshift
   ```

2. **Image Pull Issues**
   ```bash
   # Check image pull secrets
   oc get secrets -n ai-mcp-openshift | grep registry
   oc describe pod <pod-name> -n ai-mcp-openshift
   ```

3. **RBAC Permission Errors**
   ```bash
   # Check cluster role binding
   oc get clusterrolebinding openshift-ai-mcp-server -o yaml
   oc auth can-i create pods --as=system:serviceaccount:ai-mcp-openshift:openshift-ai-mcp-server
   ```

4. **Route Access Issues**
   ```bash
   # Check route configuration
   oc describe route openshift-ai-mcp-server -n ai-mcp-openshift
   
   # Test internal connectivity
   oc rsh deployment/openshift-ai-mcp-server curl localhost:8080/health
   ```

#### **Debug Commands**

```bash
# Get comprehensive pod information
oc describe pod -l app.kubernetes.io/name=openshift-ai-mcp-server -n ai-mcp-openshift

# Check resource usage
oc adm top pod -n ai-mcp-openshift

# View events
oc get events --sort-by='.lastTimestamp' -n ai-mcp-openshift

# Check security context
oc get pod -o yaml | grep -A 10 securityContext
```

### Alternative Installation Methods

#### 1. **Using oc new-app (One-Command Deployment)**

```bash
# Create new project and deploy in one command
oc new-project ai-mcp-openshift
oc new-app https://github.com/sur309/openshift-mcp-server.git \
  --name=openshift-ai-mcp-server \
  --strategy=docker

# Apply SCC and additional configurations
oc adm policy add-scc-to-user anyuid -z default -n ai-mcp-openshift
oc expose svc/openshift-ai-mcp-server
```

#### 2. **Using Helm Charts (if available)**

```bash
# Add Helm repository (if published)
helm repo add openshift-mcp-server https://your-helm-repo.com
helm repo update

# Install with Helm
helm install mcp-server openshift-mcp-server/openshift-mcp-server \
  --namespace ai-mcp-openshift \
  --create-namespace \
  --set image.tag=latest \
  --set serviceAccount.annotations."openshift\.io/scc"=anyuid
```

## 🔧 Configuration

### Container Image Management

#### Build and Push Container Image

```bash
# Build the container image
docker build -t quay.io/YOUR_ORG/ai-mcp-openshift-server:latest .

# Push to registry
docker push quay.io/YOUR_ORG/ai-mcp-openshift-server:latest

# Update deployment image
oc set image deployment/ai-mcp-openshift-server \
  inference-server=quay.io/YOUR_ORG/ai-mcp-openshift-server:latest \
  -n ai-mcp-openshift
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Inference server port | `8080` |
| `MCP_PORT` | MCP server port | `8081` |
| `MCP_PROFILE` | MCP profile to use | `cicd` |
| `LOG_LEVEL` | Logging verbosity (0-9) | `2` |
| `DEFAULT_REGISTRY` | Default container registry | `quay.io` |
| `DEFAULT_NAMESPACE` | Default deployment namespace | `ai-mcp-openshift` |
| `MODELS_PATH` | Path to ML models | `/app/models` |

### MCP Profiles

- **`full`**: All tools including CI/CD, Kubernetes management, and Helm
- **`cicd`**: Focused on CI/CD operations with essential Kubernetes tools

## 📖 Usage Examples

### 1. Set up a CI/CD Pipeline

```bash
# Add a container registry
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "registry_add",
    "arguments": {
      "name": "quay",
      "url": "quay.io",
      "username": "your-username",
      "password": "your-token"
    }
  }'

# Create a complete CI/CD pipeline
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "cicd_create_pipeline",
    "arguments": {
      "name": "my-app-pipeline",
      "git_url": "https://github.com/user/my-app.git",
      "git_branch": "main",
      "image_name": "my-app",
      "registry": "quay",
      "deploy_namespace": "my-app-prod",
      "dockerfile": "Dockerfile",
      "env_vars": {
        "NODE_ENV": "production",
        "PORT": "3000"
      }
    }
  }'
```

### 2. Manual Build and Deploy

```bash
# Build an image
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "build_image",
    "arguments": {
      "name": "my-build",
      "source_repo": "https://github.com/user/my-app.git",
      "image_name": "my-app",
      "image_tag": "v1.0.0"
    }
  }'

# Push to registry
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "registry_push",
    "arguments": {
      "source_image": "my-app:v1.0.0",
      "target_image": "quay.io/user/my-app",
      "target_tag": "v1.0.0",
      "registry": "quay"
    }
  }'

# Deploy application
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "deploy_application",
    "arguments": {
      "name": "my-app",
      "image": "quay.io/user/my-app",
      "tag": "v1.0.0",
      "namespace": "my-app-prod",
      "replicas": 3,
      "port": 3000,
      "expose_route": true
    }
  }'
```

### 3. ML Inference

```bash
# Text embedding inference
curl -X POST http://localhost:8080/infer \
  -H "Content-Type: application/json" \
  -d '{
    "inputs": ["Hello world", "OpenShift AI is great"],
    "model_name": "text_embeddings"
  }'

# Numeric inference
curl -X POST http://localhost:8080/infer \
  -H "Content-Type: application/json" \
  -d '{
    "inputs": [[1.0, 2.0, 3.0, 4.0]],
    "model_name": "simple_classifier"
  }'

# List available models
curl http://localhost:8080/models
```

## ✅ Post-Deployment Verification

### Verify MCP Server Installation

After deploying the MCP server to OpenShift, run these commands to ensure everything is working correctly:

```bash
# 1. Check pod status
oc get pods -l app.kubernetes.io/name=openshift-ai-mcp-server -n ai-mcp-openshift

# 2. Verify service account has proper SCC
oc get serviceaccount openshift-ai-mcp-server -n ai-mcp-openshift -o yaml

# 3. Check SCC assignment
oc get scc anyuid -o yaml | grep -A 10 "users:"

# 4. Test health endpoints
ROUTE_URL=$(oc get route openshift-ai-mcp-server -o jsonpath='{.spec.host}' -n ai-mcp-openshift 2>/dev/null)
if [ ! -z "$ROUTE_URL" ]; then
  echo "Testing health endpoint: https://$ROUTE_URL/health"
  curl -k -s https://$ROUTE_URL/health | jq .
else
  echo "Route not found. Testing via port-forward..."
  oc port-forward svc/openshift-ai-mcp-server 8080:8080 -n ai-mcp-openshift &
  sleep 2
  curl -s http://localhost:8080/health | jq .
  pkill -f "oc port-forward"
fi

# 5. Test MCP endpoint
curl -k -s https://$ROUTE_URL:8081/mcp/tools | jq .

# 6. Check application logs
oc logs -f deployment/openshift-ai-mcp-server --tail=50 -n ai-mcp-openshift
```

### Using the MCP CLI Tool

After deployment, you can use the included CLI tool to interact with your MCP server:

```bash
# Navigate to the CLI tool directory
cd cursor-integration/cli-tools

# Set the MCP server URL (replace with your actual route)
export MCP_SERVER_URL="https://$(oc get route openshift-ai-mcp-server-mcp -o jsonpath='{.spec.host}' -n ai-mcp-openshift)"

# Test the CLI tool
node mcp-cli.js tools

# Execute a workflow
node mcp-cli.js execute "Deploy a sample application to my OpenShift cluster"

# Analyze a prompt
node mcp-cli.js analyze "I want to build and deploy a container from my Git repository"
```

## 🔄 CI/CD Pipeline Flow

1. **Git Monitoring**: Server monitors specified Git repositories for new commits
2. **Automatic Trigger**: New commits trigger the CI/CD pipeline
3. **Image Build**: Source code is built into a container image
4. **Registry Push**: Built image is pushed to the configured registry
5. **Deployment**: Application is deployed to the specified OpenShift project
6. **Verification**: Health checks ensure successful deployment

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Git Repo      │    │   MCP Server    │    │   Inference     │
│                 │    │                 │    │   Engine        │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Commit    │ │───▶│ │  Git Watch  │ │    │ │  ML Models  │ │
│ │   Events    │ │    │ │             │ │    │ │             │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    │ ┌─────────────┐ │    │ ┌─────────────┐ │
                       │ │Image Builder│ │    │ │  FastAPI    │ │
┌─────────────────┐    │ │             │ │    │ │  Server     │ │
│Container Registry│◀───│ └─────────────┘ │    │ └─────────────┘ │
│                 │    │ ┌─────────────┐ │    └─────────────────┘
│ ┌─────────────┐ │    │ │  Deploy     │ │
│ │   Images    │ │    │ │  Manager    │ │    ┌─────────────────┐
│ │             │ │    │ │             │ │    │   OpenShift     │
│ └─────────────┘ │    │ └─────────────┘ │───▶│   Cluster       │
└─────────────────┘    └─────────────────┘    │                 │
                                              │ ┌─────────────┐ │
                                              │ │   Pods      │ │
                                              │ │   Services  │ │
                                              │ │   Routes    │ │
                                              │ └─────────────┘ │
                                              └─────────────────┘
```

## 🔧 Troubleshooting

### Common Issues

1. **Build Failures**
   ```bash
   # Check build logs
   oc logs deployment/ai-mcp-openshift-server -n ai-mcp-openshift
   ```

2. **Registry Access Issues**
   ```bash
   # Verify registry credentials
   oc get secret registry-credentials -o yaml -n ai-mcp-openshift
   ```

3. **Git Authentication**
   ```bash
   # Check Git credentials
   oc get secret git-credentials -o yaml -n ai-mcp-openshift
   ```

### Debug Commands

```bash
# Check server status
curl http://localhost:8080/health
curl http://localhost:8081/health/mcp

# List all MCP tools
curl http://localhost:8081/mcp/tools

# View server logs
oc logs -f deployment/ai-mcp-openshift-server -n ai-mcp-openshift

# Check resource usage
oc top pods -n ai-mcp-openshift
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Model Context Protocol (MCP)](https://github.com/anthropics/mcp) for the protocol specification
- [OpenShift](https://www.redhat.com/en/technologies/cloud-computing/openshift) for the container platform
- [Kubernetes](https://kubernetes.io/) for the orchestration platform

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

### 1. Deploy to OpenShift

```bash
# Clone the repository
git clone https://github.com/manusa/kubernetes-mcp-server.git
cd kubernetes-mcp-server

# Create namespace and apply manifests
oc apply -f manifests/namespace.yaml
oc apply -f manifests/rbac.yaml
oc apply -f manifests/configmap.yaml
oc apply -f manifests/secrets.yaml
oc apply -f manifests/deployment.yaml
oc apply -f manifests/service.yaml
```

### 2. Configure Secrets

```bash
# Set up container registry credentials
oc create secret generic registry-credentials \
  --from-literal=username=YOUR_REGISTRY_USERNAME \
  --from-literal=password=YOUR_REGISTRY_TOKEN \
  --from-literal=email=YOUR_EMAIL \
  -n openshift-ai-mcp

# Set up Git credentials
oc create secret generic git-credentials \
  --from-literal=username=YOUR_GIT_USERNAME \
  --from-literal=token=YOUR_GIT_TOKEN \
  -n openshift-ai-mcp

# Set up webhook secret
oc create secret generic webhook-secret \
  --from-literal=secret=YOUR_WEBHOOK_SECRET \
  -n openshift-ai-mcp
```

### 3. Build and Push Container Image

```bash
# Build the container image
docker build -t quay.io/YOUR_ORG/openshift-ai-mcp-server:latest .

# Push to registry
docker push quay.io/YOUR_ORG/openshift-ai-mcp-server:latest

# Update deployment image
oc set image deployment/openshift-ai-mcp-server \
  inference-server=quay.io/YOUR_ORG/openshift-ai-mcp-server:latest \
  -n openshift-ai-mcp
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
| `DEFAULT_NAMESPACE` | Default deployment namespace | `openshift-ai-mcp` |
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
   oc logs deployment/openshift-ai-mcp-server -n openshift-ai-mcp
   ```

2. **Registry Access Issues**
   ```bash
   # Verify registry credentials
   oc get secret registry-credentials -o yaml -n openshift-ai-mcp
   ```

3. **Git Authentication**
   ```bash
   # Check Git credentials
   oc get secret git-credentials -o yaml -n openshift-ai-mcp
   ```

### Debug Commands

```bash
# Check server status
curl http://localhost:8080/health
curl http://localhost:8081/health/mcp

# List all MCP tools
curl http://localhost:8081/mcp/tools

# View server logs
oc logs -f deployment/openshift-ai-mcp-server -n openshift-ai-mcp

# Check resource usage
oc top pods -n openshift-ai-mcp
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

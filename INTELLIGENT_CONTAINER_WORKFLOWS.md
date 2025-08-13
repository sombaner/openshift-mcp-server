# Intelligent Container Workflow Orchestration

Your OpenShift MCP Server has been enhanced with intelligent container runtime capabilities that automatically invoke the necessary tools based on natural language prompts. This allows for seamless container operations with minimal manual intervention.

## 🚀 Key Features

### 🤖 Intelligent Workflow Orchestration
- **Natural Language Processing**: Understands user intents from plain English descriptions
- **Automatic Tool Chaining**: Intelligently sequences container operations
- **Parameter Extraction**: Automatically extracts Git URLs, image names, registries, and namespaces from prompts
- **Workflow Recommendations**: Provides next steps and optimization suggestions

### 🔧 Container Runtime Operations
- **Multi-Runtime Support**: Works with both Podman and Docker
- **Red Hat UBI Validation**: Automatic Universal Base Image compliance checking
- **Security Scanning**: Built-in container security validation
- **Build Context Management**: Handles Git repositories, local directories, and remote archives

### 🗄️ Advanced Registry Management
- **Multi-Registry Support**: Docker Hub, Quay.io, GHCR, GCR, ECR, ACR
- **Authentication Management**: Secure credential storage and management
- **Image Search**: Cross-registry image discovery with ranking
- **Repository Management**: List, tag, and manage container repositories

## 🎯 Smart Commands

### Execute Workflows with Natural Language

```bash
# Build and push a container from Git repository
node mcp-cli.js execute "Build and push my app from https://github.com/user/repo.git to quay.io/user/app:latest"

# Complete CI/CD pipeline
node mcp-cli.js execute "Deploy my container application from source to production namespace"

# Security scanning
node mcp-cli.js execute "Scan my container image for security vulnerabilities"

# Multi-stage workflow
node mcp-cli.js execute "Build my Python app, run security checks, and push to registry with latest and v1.0 tags"
```

### Analyze Prompts Before Execution

```bash
# Understand what will be executed
node mcp-cli.js analyze "I want to containerize my Node.js app and deploy it to Kubernetes"

# See workflow confidence scores
node mcp-cli.js analyze "Build Docker image from GitHub and push to Quay"
```

### Manage Workflows

```bash
# List available workflows
node mcp-cli.js workflows

# List specific workflow category
node mcp-cli.js tools | grep -A5 "Workflow"
```

## 🛠️ Available Workflows

### 1. Build and Push Container
**Triggers**: "build push", "containerize deploy", "build image push registry"
**Steps**:
1. Build container image from source (with UBI validation)
2. Push to specified registry
3. Provide deployment recommendations

### 2. Complete CI/CD Pipeline
**Triggers**: "full pipeline", "complete cicd", "end-to-end deployment"
**Steps**:
1. Build container image
2. Security and compliance validation
3. Push to registry
4. Deploy to OpenShift/Kubernetes
5. Health check and monitoring setup

### 3. Security Scan Workflow
**Triggers**: "security scan", "vulnerability check", "audit container"
**Steps**:
1. Inspect container image
2. Check for vulnerabilities
3. Validate base image compliance
4. Generate security report

### 4. Registry Management
**Triggers**: "manage registry", "list images", "clean registry"
**Steps**:
1. List container images
2. Repository management
3. Tag organization
4. Cleanup recommendations

## 📋 Enhanced Container Tools

### Container Building
```bash
# Traditional way
node mcp-cli.js build https://github.com/user/app.git quay.io/user/app:latest

# Smart way
node mcp-cli.js execute "Build my app from GitHub and make it UBI compliant"
```

### Registry Operations
```bash
# Configure registries
node mcp-cli.js execute "Configure Quay.io registry with my credentials"

# Search for images
node mcp-cli.js execute "Find official Python images with Alpine Linux"

# Manage repositories
node mcp-cli.js execute "List all my repositories in Quay.io"
```

## 🔒 Security and Compliance Features

### Red Hat UBI Validation
- Automatic detection of non-UBI base images
- Suggestions for UBI alternatives
- UBI Dockerfile generation
- Enterprise compliance reporting

### Security Scanning
- Dockerfile security best practices
- Vulnerability assessment
- License compliance checking
- Supply chain security validation

## 🌐 Registry Support

### Supported Registries
- **Docker Hub** (docker.io) - Public and private repositories
- **Red Hat Quay** (quay.io) - Enterprise container registry
- **GitHub Container Registry** (ghcr.io) - GitHub-integrated registry
- **Google Container Registry** (gcr.io) - GCP-integrated registry
- **Amazon ECR** - AWS-integrated registry
- **Azure Container Registry** - Azure-integrated registry
- **Private Registries** - Custom registry support

### Registry Features
- Multi-registry authentication
- Cross-registry image search
- Repository synchronization
- Tag management and cleanup
- Security scanning integration

## 🔄 Workflow Customization

### Create Custom Workflows
```javascript
// Define custom workflow
const customWorkflow = {
  name: "My Custom Build Flow",
  description: "Custom build, test, and deploy workflow",
  keywords: ["custom", "build", "test", "deploy"],
  steps: [
    {
      tool: "container_build",
      description: "Build with custom settings",
      parameters: {
        validate_ubi: true,
        security_scan: true,
        platform: "linux/amd64"
      }
    },
    {
      tool: "container_push",
      description: "Push to multiple registries"
    }
  ]
};
```

### Workflow Conditions
- **Keyword matching**: Simple text-based triggers
- **Regex patterns**: Advanced pattern matching
- **Context analysis**: Understanding relationships between parameters
- **Confidence scoring**: Ranking workflow matches

## 📊 Usage Examples

### Example 1: Full Application Deployment
```bash
node mcp-cli.js execute "Take my Node.js app from https://github.com/mycompany/frontend.git, build it with UBI, scan for vulnerabilities, push to quay.io/mycompany/frontend:v1.2.0, and deploy to production namespace"
```

**Workflow Analysis**:
- Detects Node.js application
- Extracts Git repository URL
- Identifies target image and registry
- Determines deployment namespace
- Executes: build → scan → push → deploy

### Example 2: Security Audit
```bash
node mcp-cli.js execute "Audit my container image quay.io/mycompany/api:latest for security issues and generate a compliance report"
```

**Workflow Analysis**:
- Identifies existing container image
- Triggers security scan workflow
- Generates comprehensive audit report
- Provides remediation recommendations

### Example 3: Registry Management
```bash
node mcp-cli.js execute "Show me all my Python images in Docker Hub and Quay, and help me clean up old versions"
```

**Workflow Analysis**:
- Searches across multiple registries
- Filters by Python-related images
- Analyzes tag usage and age
- Suggests cleanup strategies

## 🔧 Configuration

### Environment Variables
```bash
# Registry credentials
export REGISTRY_USERNAME="your-username"
export REGISTRY_PASSWORD="your-token"

# MCP Server URL
export MCP_SERVER_URL="https://your-mcp-server.com"

# Container runtime preference
export CONTAINER_RUNTIME="podman"  # or "docker"
```

### Registry Configuration
```bash
# Configure multiple registries
node mcp-cli.js execute "Configure Docker Hub with username myuser"
node mcp-cli.js execute "Configure Quay.io registry for enterprise use"
node mcp-cli.js execute "Set up GitHub Container Registry with token authentication"
```

## 📈 Performance and Optimization

### Intelligent Caching
- Build context caching
- Registry authentication caching
- Workflow result caching
- Image layer optimization

### Parallel Operations
- Concurrent registry searches
- Parallel image pushes to multiple registries
- Simultaneous security scans
- Multi-platform builds

### Resource Management
- Automatic cleanup of temporary files
- Registry quota monitoring
- Build resource optimization
- Container runtime health checks

## 🚀 Getting Started

1. **Install Dependencies**
   ```bash
   cd openshift-mcp-server
   npm install
   ```

2. **Start the MCP Server**
   ```bash
   npm start
   ```

3. **Try Smart Commands**
   ```bash
   node cursor-integration/cli-tools/mcp-cli.js execute "Build my first container app"
   ```

4. **Explore Workflows**
   ```bash
   node cursor-integration/cli-tools/mcp-cli.js workflows
   ```

## 💡 Tips and Best Practices

### Prompt Engineering
- Be specific about source locations (Git URLs, local paths)
- Include target registry and image names when known
- Mention security requirements explicitly
- Specify deployment targets (namespaces, environments)

### Security Best Practices
- Use UBI base images for enterprise compliance
- Enable security scanning for all builds
- Regularly audit container dependencies
- Implement least-privilege access controls

### Performance Optimization
- Use multi-stage builds for smaller images
- Leverage build caching when possible
- Push to multiple registries in parallel
- Monitor registry quotas and limits

## 🔗 Integration

### OpenShift Integration
- Native OpenShift build configs
- S2I (Source-to-Image) support
- Operator deployment patterns
- Route and service management

### CI/CD Integration
- Jenkins pipeline integration
- GitLab CI/CD support
- GitHub Actions compatibility
- Tekton pipeline support

### Monitoring and Observability
- Build metrics collection
- Registry usage analytics
- Security scan reporting
- Performance monitoring

---

*Your OpenShift MCP Server now provides intelligent container workflow orchestration with natural language processing, making container operations more intuitive and efficient than ever before.*

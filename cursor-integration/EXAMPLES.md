# 🎯 Cursor Integration Examples

## Real-World Usage Examples

### 1. **Container Development with UBI Validation**

```bash
# Scenario: You're developing a Node.js app in Cursor
# and want to ensure it uses Red Hat UBI

# 1. Open your project in Cursor
cd my-node-app

# 2. Build with automatic UBI validation
mcp build . quay.io/myuser/my-node-app:v1.0

# Output example:
# 🔨 Building container image: quay.io/myuser/my-node-app:v1.0
# 📁 Source: /Users/dev/my-node-app
# 
# ✅ Build completed successfully!
# 📋 Build Summary:
#    • Image: quay.io/myuser/my-node-app:v1.0
#    • Duration: 2m34s
#    • Runtime: podman
#    • UBI Compliant: ⚠️  No
#    • Suggested UBI: registry.access.redhat.com/ubi8/nodejs-18:latest
# 
# 🚀 Next steps:
#    • Consider using Red Hat UBI base image
#    • UBI provides enterprise security and compliance

# 3. If you want to generate UBI Dockerfile
# (The tool suggests the optimal UBI base image)
```

### 2. **Full CI/CD Pipeline in Cursor**

```bash
# Scenario: Deploy a complete application pipeline

# 1. Add repository for monitoring
mcp repo-add https://github.com/myuser/my-web-app.git production

# 2. Trigger full auto-deployment
mcp deploy https://github.com/myuser/my-web-app.git production

# Output example:
# 🚀 Auto-deploying repository: https://github.com/myuser/my-web-app.git
# 📦 Namespace: production
# 
# ✅ Deployment completed!
# 📋 Deployment Summary:
#    • Application: my-web-app
#    • Namespace: production
#    • URL: https://my-web-app-production.apps.cluster.local

# 3. Check deployment status
mcp repo-status my-web-app

# 4. View live application
mcp repo-url my-web-app
```

### 3. **Kubernetes Operations from Cursor**

```bash
# Scenario: Debug application issues

# 1. List all pods to see what's running
mcp pods

# 2. Focus on specific namespace
mcp pods production

# 3. Check specific pod logs (you'd get pod name from previous command)
# Note: This would use the full MCP tool call
curl -X POST https://your-mcp-server.com \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "pods_log",
      "arguments": {"name": "my-web-app-xyz", "namespace": "production"}
    },
    "id": 1
  }'

# 4. Execute commands in pod for debugging
curl -X POST https://your-mcp-server.com \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call", 
    "params": {
      "name": "pods_exec",
      "arguments": {
        "name": "my-web-app-xyz",
        "namespace": "production", 
        "command": ["ps", "aux"]
      }
    },
    "id": 2
  }'
```

### 4. **Using Cursor AI with MCP Commands**

In Cursor, you can teach the AI to use your MCP tools:

```
👤 User: "I want to build my container image and make sure it's using Red Hat enterprise base images"

🤖 Cursor AI: I'll help you build a container with Red Hat UBI validation. Let me run the build command:

```bash
mcp build . quay.io/youruser/$(basename $(pwd)):latest
```

This will:
- Build your container from the current directory
- Automatically validate against Red Hat UBI standards
- Suggest UBI alternatives if needed
- Provide security and compliance benefits

Would you like me to also deploy it after building?

👤 User: "Yes, deploy it to the staging environment"

🤖 Cursor AI: Perfect! I'll deploy it to staging:

```bash
mcp deploy $(git config --get remote.origin.url) staging
```

This will create a complete CI/CD pipeline for your application in the staging namespace.
```

### 5. **Advanced Workflow: Multi-Environment Deployment**

```bash
# Scenario: Deploy same app to multiple environments

# 1. Build once with UBI validation
mcp build . quay.io/mycompany/myapp:v1.2.3

# 2. Deploy to development
mcp deploy https://github.com/mycompany/myapp.git development

# 3. Test in development, then promote to staging
mcp deploy https://github.com/mycompany/myapp.git staging

# 4. Monitor all environments
mcp pods development
mcp pods staging
mcp pods production

# 5. Check application URLs for each environment
mcp repo-url myapp  # Will show all deployed instances
```

### 6. **Container Registry Management**

```bash
# Scenario: Manage container images

# 1. List local images
mcp list-images

# 2. Push to multiple registries with different tags
mcp push quay.io/myuser/app:v1.0 latest,stable,production

# 3. Clean up old images
mcp remove-image quay.io/myuser/app:v0.9.0

# 4. Inspect image for security information
mcp inspect-image quay.io/myuser/app:v1.0
```

### 7. **Integrated Development Workflow**

```bash
# Complete development cycle in Cursor:

# 1. Start development
cd my-new-project

# 2. Write code in Cursor with AI assistance

# 3. Test build locally with UBI validation
mcp build . localhost/my-new-project:dev

# 4. If UBI issues found, AI can help fix Dockerfile:
#    Cursor AI suggests: "I see you're using 'node:18'. 
#    For enterprise compliance, consider using 
#    'registry.access.redhat.com/ubi8/nodejs-18:latest'"

# 5. Push to registry
mcp push localhost/my-new-project:dev quay.io/myuser/my-new-project:dev

# 6. Deploy to development environment
mcp deploy https://github.com/myuser/my-new-project.git development

# 7. Monitor deployment
mcp pods development | grep my-new-project

# 8. When ready, promote to production
mcp deploy https://github.com/myuser/my-new-project.git production
```

### 8. **AI-Assisted Container Optimization**

Using Cursor AI with your MCP tools:

```
👤 User: "My container build is slow and I want to optimize it for Red Hat UBI"

🤖 Cursor AI: I'll help you optimize your container build. Let me first check your current Dockerfile and build with UBI validation:

```bash
mcp build . quay.io/test/optimization:test
```

Based on the UBI validation results, I can help you:

1. **Switch to Red Hat UBI base images** for better performance and security
2. **Optimize layer caching** by reorganizing your Dockerfile
3. **Use multi-stage builds** to reduce final image size
4. **Implement security best practices** detected by the MCP validation

Let me analyze your Dockerfile and suggest improvements...

👤 User: "Show me the UBI compliance report"

🤖 Cursor AI: I'll get the detailed UBI compliance report:

```bash
# The build output will show:
# • UBI Compliant: ⚠️  No  
# • Current Base: node:18-alpine
# • Suggested UBI: registry.access.redhat.com/ubi8/nodejs-18:latest
# • Security Benefits: Enhanced security patches, FIPS compliance
# • Compliance Benefits: Enterprise support, GPL-free licensing
```

Would you like me to generate a UBI-compliant Dockerfile for you?
```

## 🎯 **Pro Tips for Cursor Integration**

### 1. **Create Custom Cursor Commands**

Add to your Cursor workspace settings:

```json
{
  "tasks": {
    "version": "2.0.0",
    "tasks": [
      {
        "label": "Build with UBI Validation",
        "type": "shell",
        "command": "mcp",
        "args": ["build", ".", "quay.io/${workspaceFolderBasename}:latest"],
        "group": "build"
      },
      {
        "label": "Deploy to Development", 
        "type": "shell",
        "command": "mcp",
        "args": ["deploy", "${input:gitUrl}", "development"],
        "group": "build"
      }
    ],
    "inputs": [
      {
        "id": "gitUrl",
        "description": "Git repository URL",
        "default": "$(git config --get remote.origin.url)"
      }
    ]
  }
}
```

### 2. **Keyboard Shortcuts**

Add to Cursor keybindings:

```json
[
  {
    "key": "cmd+shift+b",
    "command": "workbench.action.terminal.sendSequence",
    "args": {
      "text": "mcp build . quay.io/$(basename $(pwd)):latest\n"
    }
  },
  {
    "key": "cmd+shift+d", 
    "command": "workbench.action.terminal.sendSequence",
    "args": {
      "text": "mcp deploy $(git config --get remote.origin.url) development\n"
    }
  }
]
```

### 3. **AI Context Training**

Train Cursor AI by adding this to your project's `.cursor-rules`:

```
This project uses OpenShift MCP Server for container builds and deployments.

Available commands:
- mcp build <source> <image> - Build container with Red Hat UBI validation
- mcp deploy <repo> <namespace> - Auto-deploy to OpenShift
- mcp pods [namespace] - List Kubernetes pods
- mcp tools - Show all available tools

Always suggest Red Hat UBI base images for enterprise compliance.
Use the MCP tools for any container or Kubernetes operations.
```

This integration gives you the full power of your OpenShift MCP server directly within Cursor! 🚀


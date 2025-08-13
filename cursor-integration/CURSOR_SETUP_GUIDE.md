# 🚀 Cursor Integration Setup Guide

## Overview

This guide shows you how to integrate your OpenShift MCP Server with Cursor IDE for seamless container building, CI/CD automation, and Kubernetes management.

## ✅ Available Integration Methods

### 1. **CLI Tool Integration** (Recommended - Ready Now!)

Use the MCP CLI tool directly in Cursor's terminal.

#### Quick Setup:
```bash
# Add to your shell profile (~/.zshrc or ~/.bashrc)
alias mcp="node /Users/sureshgaikwad/openshift-mcp-server/cursor-integration/cli-tools/mcp-cli.js"

# Reload shell
source ~/.zshrc
```

#### Usage in Cursor:
```bash
# View all available tools
mcp tools

# Build container with Red Hat UBI validation
mcp build . quay.io/myuser/myapp:latest

# Auto-deploy current project
mcp deploy https://github.com/myuser/myproject.git my-namespace

# Kubernetes operations
mcp pods my-namespace
```

### 2. **VS Code Extension** (Advanced Integration)

Install the custom extension for deep Cursor integration.

#### Setup:
```bash
cd /Users/sureshgaikwad/openshift-mcp-server/cursor-integration/vscode-extension
npm install -g vsce
vsce package
```

#### Features:
- Right-click on Dockerfile → "Build Container with UBI Validation"
- Command Palette: "OpenShift MCP: Auto-Deploy Repository"
- Automatic UBI compliance checking
- Integrated output panel with build logs

### 3. **Direct HTTP API** (For Advanced Users)

Use curl or HTTP requests directly from Cursor terminal.

#### Example:
```bash
# List all tools
curl -X POST https://your-mcp-server.com \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": 1}'

# Build container
curl -X POST https://your-mcp-server.com \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0", 
    "method": "tools/call", 
    "params": {
      "name": "container_build",
      "arguments": {
        "source": ".",
        "image_name": "quay.io/user/app:latest",
        "validate_ubi": true
      }
    }, 
    "id": 2
  }'
```

## 🎯 **Recommended Workflow in Cursor**

### 1. **Container Development Workflow**

```bash
# 1. Open your project in Cursor
# 2. Build with UBI validation
mcp build . quay.io/myuser/myapp:v1.0

# 3. If non-UBI detected, get suggestions
# Tool will suggest Red Hat UBI alternatives

# 4. Push to registry
mcp push quay.io/myuser/myapp:v1.0

# 5. Deploy to OpenShift
mcp deploy https://github.com/myuser/myapp.git production
```

### 2. **CI/CD Integration**

```bash
# Add repository for monitoring
mcp repo-add https://github.com/myuser/myapp.git production

# Check build status
mcp repo-status myapp

# Manual build trigger
mcp repo-build myapp

# Get application URL
mcp repo-url myapp
```

### 3. **Kubernetes Management**

```bash
# List pods across namespaces
mcp pods

# List pods in specific namespace
mcp pods production

# Check resource usage
mcp resources list apps/v1 Deployment

# View configuration
mcp config-view
```

## 🛠️ **Available Tools (31 Total)**

### 🐳 **Container Tools (5)**
- `container_build` - Build with Red Hat UBI validation
- `container_push` - Push to registries
- `container_list` - List local images
- `container_remove` - Clean up images
- `container_inspect` - Analyze image details

### 🚀 **CI/CD Tools (9)**
- `repo_add` - Add repository for monitoring
- `repo_auto_deploy` - Fully automated deployment
- `repo_build` - Manual build trigger
- `repo_deploy` - Deploy to OpenShift
- `repo_status` - Check pipeline status
- And 4 more...

### ☸️ **Kubernetes Tools (14)**
- `pods_list` - List all pods
- `namespace_create` - Create namespaces
- `resources_*` - Manage any Kubernetes resource
- And 11 more...

### ⚙️ **Other Tools (3)**
- `cicd_status` - System status
- `configuration_view` - View kubeconfig
- `projects_list` - List OpenShift projects

## 🔧 **Cursor AI Integration Tips**

### 1. **Use Natural Language with Cursor AI**

Train Cursor AI to use your MCP tools:

```
"Build a container image from the current directory and validate it uses Red Hat UBI base images"

"Deploy this repository to the production namespace in OpenShift"

"Show me all running pods in the development namespace"
```

### 2. **Create Cursor AI Rules**

Add to your Cursor settings:

```json
{
  "cursor.cpp.disabledLanguageFeatures": [],
  "cursor.general.enableAI": true,
  "cursor.general.customCommands": [
    {
      "name": "Build with UBI",
      "command": "mcp build . quay.io/${workspace}/app:latest"
    },
    {
      "name": "Deploy to OpenShift", 
      "command": "mcp deploy ${git_origin} ${workspace}"
    }
  ]
}
```

### 3. **Integrate with Cursor Workflows**

Create keyboard shortcuts in Cursor:

- **Cmd+Shift+B**: Build container with UBI validation
- **Cmd+Shift+D**: Auto-deploy to OpenShift
- **Cmd+Shift+K**: List Kubernetes pods

## 🎉 **Benefits of Integration**

### ✅ **Red Hat UBI Validation**
- Automatic detection of non-UBI base images
- Suggestions for Red Hat UBI alternatives
- Enterprise security and compliance

### ✅ **Seamless CI/CD**
- One-command deployment to OpenShift
- Automatic manifest generation
- Build and deployment automation

### ✅ **Kubernetes Management**
- Full cluster visibility from Cursor
- Resource management and monitoring
- Pod logs and execution

### ✅ **Developer Productivity**
- No context switching between tools
- AI-assisted container operations
- Integrated development workflow

## 🚀 **Get Started Now**

1. **Set up the alias** (30 seconds):
   ```bash
   echo 'alias mcp="node /Users/sureshgaikwad/openshift-mcp-server/cursor-integration/cli-tools/mcp-cli.js"' >> ~/.zshrc
   source ~/.zshrc
   ```

2. **Test it** (30 seconds):
   ```bash
   mcp tools
   ```

3. **Build your first container with UBI validation**:
   ```bash
   mcp build . quay.io/myuser/myapp:latest
   ```

**You're ready to go!** 🎉

The integration gives you all the power of your OpenShift MCP server directly within Cursor's development environment.



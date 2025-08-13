# 🔧 VS Code Integration for OpenShift MCP Server

## Overview

VS Code doesn't natively support MCP protocol like Claude Desktop, but we can integrate using a custom extension that provides seamless access to all MCP server capabilities.

## 🚀 **Method 1: VS Code Extension** (Recommended)

### Installation Steps

1. **Build the Extension:**
```bash
cd /Users/sureshgaikwad/openshift-mcp-server/cursor-integration/vscode-extension
npm install -g vsce
vsce package
```

2. **Install in VS Code:**
```bash
# This creates openshift-mcp-cursor-1.0.0.vsix
code --install-extension openshift-mcp-cursor-1.0.0.vsix
```

3. **Configure Extension:**

Add to your VS Code `settings.json`:

```json
{
  "openshift-mcp.serverUrl": "https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
}
```

### Available Commands

Once installed, you can access these commands via Command Palette (`Cmd+Shift+P`):

- **OpenShift MCP: Build Container with UBI Validation**
- **OpenShift MCP: Auto-Deploy Repository** 
- **OpenShift MCP: List Available Tools**
- **OpenShift MCP: List Kubernetes Pods**

### Context Menu Integration

- Right-click on any `Dockerfile` → **"Build Container with UBI Validation"**

## 🛠️ **Method 2: Terminal Integration** (Quick Setup)

### 1. Add MCP CLI Alias

```bash
# Add to your shell profile
echo 'alias mcp="node /Users/sureshgaikwad/openshift-mcp-server/cursor-integration/cli-tools/mcp-cli.js"' >> ~/.zshrc
source ~/.zshrc
```

### 2. VS Code Tasks Configuration

Create `.vscode/tasks.json` in your project:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "🔨 Build Container (UBI Validated)",
            "type": "shell",
            "command": "mcp",
            "args": ["build", ".", "quay.io/${workspaceFolderBasename}:latest"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared",
                "showReuseMessage": true,
                "clear": false
            },
            "problemMatcher": []
        },
        {
            "label": "🚀 Deploy to OpenShift",
            "type": "shell", 
            "command": "mcp",
            "args": ["deploy", "${input:gitUrl}", "${input:namespace}"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": []
        },
        {
            "label": "📋 List Available Tools",
            "type": "shell",
            "command": "mcp", 
            "args": ["tools"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "🐳 List Kubernetes Pods",
            "type": "shell",
            "command": "mcp",
            "args": ["pods", "${input:namespace}"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always", 
                "focus": false,
                "panel": "shared"
            }
        }
    ],
    "inputs": [
        {
            "id": "gitUrl",
            "description": "Git Repository URL",
            "default": "",
            "type": "promptString"
        },
        {
            "id": "namespace", 
            "description": "OpenShift Namespace",
            "default": "development",
            "type": "promptString"
        }
    ]
}
```

### 3. VS Code Keybindings

Create `.vscode/keybindings.json`:

```json
[
    {
        "key": "cmd+shift+b",
        "command": "workbench.action.tasks.runTask",
        "args": "🔨 Build Container (UBI Validated)"
    },
    {
        "key": "cmd+shift+d",
        "command": "workbench.action.tasks.runTask", 
        "args": "🚀 Deploy to OpenShift"
    },
    {
        "key": "cmd+shift+t",
        "command": "workbench.action.tasks.runTask",
        "args": "📋 List Available Tools"
    },
    {
        "key": "cmd+shift+k",
        "command": "workbench.action.tasks.runTask",
        "args": "🐳 List Kubernetes Pods"
    }
]
```

## 🎯 **Method 3: HTTP REST Client** (For API Testing)

### Install REST Client Extension

```bash
code --install-extension humao.rest-client
```

### Create `mcp-requests.http`:

```http
### List All Available Tools
POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "params": {},
  "id": 1
}

### Build Container with UBI Validation
POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "container_build",
    "arguments": {
      "source": "https://github.com/myuser/myapp.git",
      "image_name": "quay.io/myuser/myapp:latest",
      "validate_ubi": true,
      "security_scan": true
    }
  },
  "id": 2
}

### Auto-Deploy Repository
POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "repo_auto_deploy",
    "arguments": {
      "url": "https://github.com/myuser/myapp.git",
      "namespace": "production"
    }
  },
  "id": 3
}

### List Pods in Namespace
POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "pods_list_in_namespace",
    "arguments": {
      "namespace": "production"
    }
  },
  "id": 4
}
```

## 🚀 **Quick Start Guide**

### For Extension Method:
1. Build and install the extension
2. Reload VS Code
3. Use Command Palette or right-click context menus
4. Enjoy integrated container building with UBI validation!

### For Terminal Method:
1. Set up the alias: `echo 'alias mcp="node /path/to/mcp-cli.js"' >> ~/.zshrc`
2. Add tasks.json to your project
3. Use `Cmd+Shift+P` → "Tasks: Run Task"
4. Or use the keyboard shortcuts

### For REST Client Method:
1. Install REST Client extension
2. Create the `.http` file
3. Click "Send Request" above each request
4. Perfect for API testing and exploration

## 🎯 **Recommended Workflow**

1. **Development**: Use Terminal integration with tasks for seamless workflow
2. **Testing**: Use REST Client for API exploration
3. **Production**: Use Extension for integrated UI experience

## 🔧 **Configuration Options**

### Extension Settings

Add to your `settings.json`:

```json
{
  "openshift-mcp.serverUrl": "https://your-mcp-server.com",
  "openshift-mcp.defaultNamespace": "development",
  "openshift-mcp.autoValidateUBI": true,
  "openshift-mcp.showBuildLogs": true
}
```

### Environment Variables

```bash
export MCP_SERVER_URL="https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
export DEFAULT_REGISTRY="quay.io/yourusername"
export DEFAULT_NAMESPACE="development"
```

## ✨ **Benefits**

### 🐳 **Container Operations**
- Build with automatic Red Hat UBI validation
- Push to multiple registries
- Security scanning and compliance checks

### 🚀 **CI/CD Integration**
- One-click deployment to OpenShift
- Automatic manifest generation
- Pipeline monitoring and status

### ☸️ **Kubernetes Management**
- Full cluster visibility
- Pod management and logs
- Resource operations

### 🛡️ **Security & Compliance**
- Red Hat UBI validation
- Security best practices
- Enterprise-grade compliance

All methods give you access to the **31 available MCP tools** for complete OpenShift and container management! 🎉



# 🤖 GitHub Copilot Integration Guide

This guide walks you through integrating your deployed OpenShift AI MCP server with GitHub Copilot for automated CI/CD workflows.

## 🌐 **Server Endpoints**

Your OpenShift AI MCP server is deployed with these endpoints:

| Service | URL | Purpose |
|---------|-----|---------|
| **Inference API** | `https://openshift-ai-mcp-server-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com` | ML inference and health checks |
| **MCP Server** | `https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com` | Model Context Protocol for CI/CD tools |

## 🛠️ **VS Code + Copilot Setup**

### **Step 1: Install Required Extensions**

```bash
# Install VS Code extensions
code --install-extension ms-vscode.vscode-copilot
code --install-extension ms-vscode.vscode-copilot-chat
code --install-extension modelcontextprotocol.mcp-client
```

### **Step 2: Configure MCP Server Connection**

The repository already includes VS Code configuration files:

#### **`.vscode/settings.json`** - Main VS Code settings
```json
{
    "mcp.servers": {
        "openshift-ai": {
            "url": "https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com",
            "profile": "cicd",
            "enabled": true
        }
    },
    "mcp.defaultServer": "openshift-ai",
    "copilot.advanced": {
        "mcp.integration": true,
        "mcp.server": "openshift-ai",
        "autoTrigger": true
    }
}
```

#### **`.vscode/mcp-config.json`** - Detailed MCP configuration
```json
{
    "mcpServers": {
        "openshift-ai": {
            "name": "OpenShift AI MCP Server",
            "url": "https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com",
            "type": "http",
            "capabilities": [
                "git_repository_management",
                "container_image_building", 
                "deployment_automation",
                "kubernetes_operations"
            ]
        }
    }
}
```

### **Step 3: Custom Copilot Commands**

Use these custom commands in VS Code:

| Command | Purpose | Usage |
|---------|---------|-------|
| `/deploy` | Deploy current project to OpenShift | `@copilot /deploy` |
| `/build` | Build container image | `@copilot /build` |
| `/watch-repo` | Set up repository monitoring | `@copilot /watch-repo` |
| `/status` | Check deployment status | `@copilot /status` |

### **Step 4: Test the Integration**

1. **Open VS Code in this repository**:
   ```bash
   code .
   ```

2. **Open Copilot Chat** (`Ctrl/Cmd + Shift + I`)

3. **Test MCP server connection**:
   ```
   @copilot Can you connect to the OpenShift AI MCP server?
   ```

4. **Test CI/CD capabilities**:
   ```
   @copilot /watch-repo
   ```

## 🔧 **Manual MCP Testing**

### **Test Server Health**
```bash
# Test inference endpoint
curl -s https://openshift-ai-mcp-server-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com/health

# Test MCP endpoint (JSON-RPC)
curl -X POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}'
```

### **Test Available Tools**
```bash
# List available MCP tools
curl -X POST https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list", 
    "params": {},
    "id": 2
  }'
```

## 🚀 **Automated CI/CD Workflows**

### **Workflow 1: Auto-Deploy on Commit**

When you commit to `main` branch, the MCP server can:

1. **Detect the commit** (Git watcher)
2. **Build container image** (Docker-in-Docker on OpenShift)
3. **Push to registry** (Quay.io/Docker Hub)
4. **Deploy to OpenShift** (Update deployment manifest)

### **Workflow 2: Repository Monitoring**

Set up continuous monitoring:

```
@copilot /watch-repo
```

This will:
- Monitor your Git repository for changes
- Trigger builds automatically
- Deploy to the `ai-mcp-openshift` namespace

### **Workflow 3: Manual Deployment**

Deploy current project:

```
@copilot /deploy
```

This will:
- Build a container image from current code
- Push to configured registry
- Deploy to OpenShift with proper manifests

## 🛠️ **Troubleshooting**

### **Issue: MCP Server Not Responding**

1. **Check pod status**:
   ```bash
   oc get pods -n ai-mcp-openshift
   oc logs deployment/openshift-ai-mcp-server -n ai-mcp-openshift
   ```

2. **Test internal connectivity**:
   ```bash
   oc port-forward deployment/openshift-ai-mcp-server -n ai-mcp-openshift 8081:8081
   curl http://localhost:8081/health
   ```

3. **Check routes**:
   ```bash
   oc get routes -n ai-mcp-openshift
   ```

### **Issue: Copilot Not Connecting**

1. **Restart VS Code**
2. **Check MCP extension logs** in VS Code output panel
3. **Verify network connectivity** to OpenShift routes

### **Issue: Authentication Errors**

The current setup uses **no authentication** for simplicity. For production:

1. **Add OpenShift token authentication**:
   ```bash
   export OPENSHIFT_TOKEN=$(oc whoami -t)
   ```

2. **Update VS Code settings** to include token authentication

## 📝 **Example Usage**

### **Basic CI/CD Setup**

1. Open this repository in VS Code
2. Open Copilot Chat
3. Run: `@copilot /watch-repo`
4. Make changes to your code
5. Commit and push
6. Watch automatic deployment!

### **Manual Build and Deploy**

```
@copilot Please help me:
1. Build a container image for this project
2. Push it to Quay.io
3. Deploy it to OpenShift in the ai-mcp-openshift namespace
```

### **Check Status**

```
@copilot /status
```

This will show:
- Current deployments in ai-mcp-openshift namespace
- Pod status and health
- Available routes and services

## 🔗 **Useful Links**

- **Inference API**: https://openshift-ai-mcp-server-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com/health
- **MCP Server**: https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
- **OpenShift Console**: Check your cluster's console for deployment details
- **GitHub Repository**: This repository for latest code and documentation

## 🎉 **Next Steps**

1. **Test the integration** with the steps above
2. **Customize workflows** for your specific needs
3. **Add authentication** for production use
4. **Monitor and scale** your deployments

Your OpenShift AI MCP server is ready for GitHub Copilot integration! 🚀

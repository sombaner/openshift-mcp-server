# 🤖 GitHub Copilot Integration Guide - Multi-Repository CI/CD

This guide walks you through integrating your deployed OpenShift AI MCP server with GitHub Copilot for **multi-repository automated CI/CD workflows**. The MCP server can now monitor and deploy **any Git repository** to **any OpenShift namespace/project**.

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
| `/add-repo` | Add any Git repository for CI/CD monitoring | `@copilot /add-repo` |
| `/list-repos` | List all monitored repositories | `@copilot /list-repos` |
| `/build` | Trigger build for a specific repository | `@copilot /build` |
| `/deploy` | Deploy repository to its configured namespace | `@copilot /deploy` |
| `/status` | Get repository CI/CD pipeline status | `@copilot /status` |
| `/create-namespace` | Create new OpenShift namespace/project | `@copilot /create-namespace` |
| `/cicd-status` | Get overall system status | `@copilot /cicd-status` |

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

## 🚀 **Multi-Repository CI/CD Workflows**

### **Workflow 1: Add Any Repository**

Add any Git repository for monitoring and deployment:

```
@copilot Please add the repository https://github.com/user/my-app.git for CI/CD monitoring. 
Deploy it to the 'my-app-prod' namespace.
```

This will:
- Configure the repository for monitoring
- Set up build and deployment pipelines
- Use the specified OpenShift namespace

### **Workflow 2: Monitor Multiple Repositories**

List and manage multiple repositories:

```
@copilot /list-repos
```

Shows all configured repositories with their:
- Git URLs and branches
- Target namespaces
- Container registries
- Build/deployment status

### **Workflow 3: Cross-Project Deployment**

Deploy any repository to any namespace:

```
@copilot Please deploy the 'my-frontend' repository to the 'staging' namespace 
instead of its default 'production' namespace.
```

### **Workflow 4: Create New Projects**

Create new OpenShift projects/namespaces:

```
@copilot /create-namespace
# Creates a new namespace for deployments
```

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

### **Multi-Repository CI/CD Setup**

1. **Add Multiple Repositories**:
   ```
   @copilot Please add these repositories for CI/CD monitoring:
   - https://github.com/company/frontend.git → deploy to 'frontend-prod' namespace
   - https://github.com/company/backend.git → deploy to 'backend-prod' namespace  
   - https://github.com/company/database.git → deploy to 'data-services' namespace
   ```

2. **Cross-Project Deployment**:
   ```
   @copilot Please:
   1. Build the 'frontend' repository
   2. Push the image to quay.io/company/frontend
   3. Deploy it to the 'staging' namespace for testing
   ```

3. **System Overview**:
   ```
   @copilot /cicd-status
   ```
   
   This shows:
   - Total monitored repositories
   - Status breakdown (building, deploying, deployed)
   - Available actions for each repository

4. **Repository-Specific Status**:
   ```
   @copilot /status
   # Shows detailed pipeline status for current/specified repository
   ```

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

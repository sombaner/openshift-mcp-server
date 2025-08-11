# VS Code MCP Integration with GitHub Copilot

## Overview
Configure VS Code to use the OpenShift AI MCP Server for automated CI/CD workflows triggered by code commits, integrated with GitHub Copilot.

## Architecture

```
VS Code + Copilot → MCP Client → OpenShift AI MCP Server → OpenShift/K8s API
                                         ↓
                              Automated CI/CD Pipeline
                                         ↓
                              Git Repo → Build → Deploy
```

## Prerequisites

### VS Code Extensions
1. **MCP Extension** (if available) or custom MCP client
2. **GitHub Copilot** extension
3. **OpenShift Toolkit** extension
4. **Kubernetes** extension

### OpenShift Deployment
- MCP Server deployed on OpenShift (see DEPLOYMENT.md)
- External route configured and accessible

## Configuration

### 1. Get MCP Server Endpoint

After deployment, get the MCP server URL:
```bash
# Get MCP endpoint URL
MCP_URL=$(oc get route ai-mcp-openshift-server-mcp -o jsonpath='{.spec.host}')
echo "MCP Server URL: https://$MCP_URL"

# Get inference endpoint URL  
INFERENCE_URL=$(oc get route ai-mcp-openshift-server -o jsonpath='{.spec.host}')
echo "Inference URL: https://$INFERENCE_URL"
```

### 2. VS Code MCP Configuration

Create `.vscode/mcp-config.json` in your project:
```json
{
  "mcpServers": {
    "openshift-ai": {
      "name": "OpenShift AI MCP Server",
      "url": "https://your-mcp-route-url",
      "type": "http",
      "capabilities": [
        "git_repository_management",
        "container_image_building", 
        "deployment_automation",
        "kubernetes_operations"
      ],
      "authentication": {
        "type": "bearer",
        "token": "${OPENSHIFT_TOKEN}"
      },
      "timeout": 300,
      "retries": 3
    }
  },
  "profiles": {
    "cicd": {
      "server": "openshift-ai",
      "tools": [
        "git_add_repository_simple",
        "git_list_repositories_simple"
      ]
    }
  }
}
```

### 3. Environment Variables

Create `.vscode/settings.json`:
```json
{
  "mcp.servers": {
    "openshift-ai": {
      "url": "https://your-mcp-route-url",
      "profile": "cicd"
    }
  },
  "openshift.token": "${env:OPENSHIFT_TOKEN}",
  "copilot.advanced": {
    "mcp.integration": true,
    "mcp.server": "openshift-ai"
  }
}
```

### 4. Authentication Setup

Create OpenShift service account token:
```bash
# Create service account for VS Code
oc create sa vscode-mcp-client -n ai-mcp-openshift

# Create token
oc create token vscode-mcp-client -n ai-mcp-openshift --duration=8760h

# Grant necessary permissions
oc adm policy add-role-to-user edit system:serviceaccount:ai-mcp-openshift:vscode-mcp-client -n ai-mcp-openshift
```

Set environment variable:
```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export OPENSHIFT_TOKEN="your-service-account-token"
```

## GitHub Copilot Integration

### 1. Copilot Chat Commands

Configure custom Copilot commands for CI/CD operations:

Create `.vscode/copilot-commands.json`:
```json
{
  "commands": {
    "/deploy": {
      "description": "Deploy current project to OpenShift",
      "handler": "mcp",
      "server": "openshift-ai",
      "tool": "deploy_application",
      "context": ["workspace", "git"]
    },
    "/build": {
      "description": "Build container image for current project", 
      "handler": "mcp",
      "server": "openshift-ai",
      "tool": "build_container_image",
      "context": ["workspace", "git", "dockerfile"]
    },
    "/watch-repo": {
      "description": "Set up repository monitoring for CI/CD",
      "handler": "mcp", 
      "server": "openshift-ai",
      "tool": "git_add_repository_simple",
      "context": ["git"]
    },
    "/pipeline-status": {
      "description": "Check CI/CD pipeline status",
      "handler": "mcp",
      "server": "openshift-ai", 
      "tool": "get_deployment_status",
      "context": ["workspace"]
    }
  }
}
```

### 2. Copilot Workflow Triggers

Configure automatic triggers in `.vscode/copilot-workflows.json`:
```json
{
  "workflows": {
    "auto-deploy": {
      "trigger": {
        "type": "git-commit",
        "branch": ["main", "develop"],
        "paths": ["src/**", "Dockerfile", "manifests/**"]
      },
      "actions": [
        {
          "type": "mcp-call",
          "server": "openshift-ai",
          "tool": "git_add_repository_simple", 
          "args": {
            "url": "${git.remoteUrl}",
            "branch": "${git.branch}"
          }
        }
      ],
      "confirmation": "auto"
    },
    "manual-build": {
      "trigger": {
        "type": "command",
        "command": "/build"
      },
      "actions": [
        {
          "type": "mcp-call",
          "server": "openshift-ai",
          "tool": "build_container_image",
          "args": {
            "source": "${workspace.path}",
            "registry": "quay.io",
            "image_name": "${workspace.name}"
          }
        }
      ]
    }
  }
}
```

## Usage Examples

### 1. Setup Repository Monitoring

In VS Code, open Copilot Chat and type:
```
@copilot /watch-repo

# Copilot will call the MCP server to add the current Git repository for monitoring
```

### 2. Deploy Application

```
@copilot /deploy

# Copilot will trigger the CI/CD pipeline through the MCP server
```

### 3. Check Pipeline Status

```
@copilot /pipeline-status

# Copilot will query the MCP server for current deployment status
```

### 4. Manual Build

```
@copilot /build

# Copilot will trigger a container image build
```

## Automated Workflows

### Commit-Triggered Deployment

When you commit code to monitored branches:

1. **VS Code** detects git commit
2. **Copilot** triggers workflow
3. **MCP Client** calls OpenShift AI MCP Server
4. **MCP Server** initiates CI/CD pipeline:
   - Pulls latest code
   - Builds container image
   - Pushes to registry
   - Updates OpenShift deployment
   - Reports status back

### Real-time Feedback

Copilot provides real-time feedback:
```
🔄 Building container image...
✅ Image built: quay.io/myapp:abc123
🚀 Deploying to OpenShift...
✅ Deployment successful: https://myapp-route.cluster.local
```

## Testing the Integration

### 1. Test MCP Connection

```bash
# Test from VS Code terminal
curl -X POST https://your-mcp-route/tools/git_list_repositories_simple \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 2. Test Copilot Commands

In VS Code:
1. Open Command Palette (`Cmd+Shift+P`)
2. Type "Copilot: Open Chat"
3. Test: `@copilot /watch-repo`

### 3. Test Automated Workflow

1. Make a code change
2. Commit to main branch
3. Watch Copilot chat for deployment updates

## Troubleshooting

### MCP Connection Issues

```bash
# Check MCP server health
curl -k https://your-mcp-route/health/mcp

# Verify authentication
curl -k -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  https://your-mcp-route/health/mcp
```

### Copilot Integration Issues

1. **Check VS Code logs**: `View → Output → Copilot`
2. **Verify MCP config**: Check `.vscode/mcp-config.json` syntax
3. **Token expiry**: Regenerate OpenShift token if expired

### Pipeline Failures

1. **Check MCP server logs**: `oc logs -l app.kubernetes.io/name=ai-mcp-openshift-server`
2. **Verify RBAC**: Ensure service account has necessary permissions
3. **Network connectivity**: Test from within cluster

## Advanced Configuration

### Custom Tools

Extend MCP server with custom tools by modifying the server configuration:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-server-config
data:
  custom_tools.json: |
    {
      "tools": [
        {
          "name": "custom_deploy",
          "description": "Custom deployment with specific parameters",
          "parameters": {
            "namespace": "string",
            "replicas": "number",
            "strategy": "string"
          }
        }
      ]
    }
```

### Multi-Cluster Support

Configure multiple OpenShift clusters:
```json
{
  "mcpServers": {
    "openshift-dev": {
      "url": "https://dev-mcp-route",
      "profile": "development"
    },
    "openshift-prod": {
      "url": "https://prod-mcp-route", 
      "profile": "production"
    }
  }
}
```

## Security Best Practices

1. **Token Rotation**: Regularly rotate service account tokens
2. **Least Privilege**: Grant minimal required RBAC permissions
3. **Network Policies**: Restrict MCP server network access
4. **Audit Logging**: Enable OpenShift audit logs for MCP operations
5. **TLS**: Always use HTTPS for MCP communication

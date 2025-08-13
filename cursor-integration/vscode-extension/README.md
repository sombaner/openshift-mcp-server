# OpenShift MCP VS Code Extension

## Overview

This VS Code extension integrates your OpenShift MCP Server directly into the VS Code development environment, providing seamless access to container building, CI/CD automation, and Kubernetes management.

## Features

- ✅ **Container Building with Red Hat UBI Validation**
- ✅ **Auto-Deployment to OpenShift**  
- ✅ **Kubernetes Pod Management**
- ✅ **Tool Discovery and Listing**
- ✅ **Integrated Output Panel with Build Logs**
- ✅ **Context Menu Integration**

## Installation

### Method 1: Package and Install

```bash
# Build the extension
cd vscode-extension
npm install -g vsce
vsce package

# Install in VS Code
code --install-extension openshift-mcp-cursor-1.0.0.vsix
```

### Method 2: Development Mode

```bash
# Open in VS Code
code .

# Press F5 to launch Extension Development Host
```

## Configuration

Add to your VS Code settings (`settings.json`):

```json
{
  "openshift-mcp.serverUrl": "https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
}
```

## Available Commands

### Command Palette (`Cmd+Shift+P`)

- **OpenShift MCP: Build Container with UBI Validation**
- **OpenShift MCP: Auto-Deploy Repository**
- **OpenShift MCP: List Available Tools**
- **OpenShift MCP: List Kubernetes Pods**

### Context Menu

- Right-click on `Dockerfile` → **"Build Container with UBI Validation"**

## Usage Examples

### 1. Build Container with UBI Validation

1. Open a project with a Dockerfile
2. `Cmd+Shift+P` → "OpenShift MCP: Build Container"
3. Enter image name (e.g., `quay.io/user/app:latest`)
4. Watch build progress in Output panel
5. Get UBI compliance recommendations

### 2. Auto-Deploy Repository

1. `Cmd+Shift+P` → "OpenShift MCP: Auto-Deploy Repository"
2. Enter Git URL and namespace
3. Monitor deployment progress
4. Get live application URL

### 3. Kubernetes Management

1. `Cmd+Shift+P` → "OpenShift MCP: List Kubernetes Pods"
2. Enter namespace (optional)
3. View pod status and details
4. Access logs and execute commands

## Output Panel

All operations show detailed output in the **"OpenShift MCP"** output panel:

```
🔨 Building container image: quay.io/user/app:latest
📁 Source: /Users/dev/my-app

✅ Build completed successfully!
📋 Build Summary:
   • Image: quay.io/user/app:latest
   • Duration: 2m34s  
   • Runtime: podman
   • UBI Compliant: ⚠️  No
   • Suggested UBI: registry.access.redhat.com/ubi8/nodejs-18:latest

🚀 Next steps:
   • Consider using Red Hat UBI base image
   • UBI provides enterprise security and compliance
```

## Extension Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `openshift-mcp.serverUrl` | MCP Server URL | `https://...` |

## Requirements

- VS Code 1.60.0 or higher
- Node.js for CLI tools (optional)
- Access to OpenShift MCP Server

## Known Issues

- Requires network access to MCP server
- Self-signed certificates may need configuration

## Release Notes

### 1.0.0

- Initial release
- Container building with UBI validation
- Auto-deployment capabilities
- Kubernetes pod management
- Tool discovery

## Development

### Building

```bash
npm install
vsce package
```

### Testing

```bash
# Open in VS Code and press F5 for Extension Development Host
code .
```

## License

ISC

## Support

For issues and feature requests, please use the GitHub repository.


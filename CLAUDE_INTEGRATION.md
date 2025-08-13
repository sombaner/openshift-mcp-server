# Claude Desktop Integration with OpenShift MCP Server

## Updated Configuration

The OpenShift MCP Server now includes enhanced container tools with Red Hat UBI validation. The Claude Desktop configuration has been updated to reflect the new capabilities.

## Current Configuration

Your Claude Desktop is now configured to connect directly to the deployed MCP server at:
```
https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com
```

## New Container Tools Available

The MCP server now includes comprehensive container management tools:

### 1. **container_build** - Enhanced Container Building
- **Red Hat UBI Validation**: Automatically validates if your Dockerfile uses Red Hat Universal Base Images
- **Security Scanning**: Performs security validation on Dockerfiles
- **UBI Dockerfile Generation**: Can automatically generate UBI-compliant Dockerfiles
- **Multi-source Support**: Build from Git repositories, local directories, or remote archives
- **Platform Support**: Multi-platform builds (linux/amd64, linux/arm64)

**Example Usage:**
```
Build a container image with UBI validation:
- source: https://github.com/user/my-app.git
- image_name: quay.io/myuser/my-app:latest
- validate_ubi: true
- generate_ubi_dockerfile: true (if non-UBI base detected)
```

### 2. **container_push** - Registry Management
- **Multi-registry Support**: Push to Quay.io, Docker Hub, GitHub Container Registry, etc.
- **Authentication**: Supports username/password or environment variable authentication
- **Multi-tag Support**: Push multiple tags simultaneously
- **TLS Configuration**: Support for private registries

### 3. **container_list** - Local Image Management
- **Image Discovery**: List all local container images
- **Filtering**: Filter by registry, name patterns, or tags
- **Format Options**: Table or JSON output for different use cases

### 4. **container_remove** - Cleanup Management
- **Safe Removal**: Remove unused container images
- **Force Options**: Override safety checks when needed
- **Pruning**: Automatically clean up unused parent images

### 5. **container_inspect** - Image Analysis
- **Detailed Metadata**: View comprehensive image information
- **Layer Analysis**: Understand image composition
- **Security Information**: View security-related image details

## Red Hat UBI Benefits

When the MCP server detects non-UBI base images, it will:

1. **Suggest Red Hat UBI Alternatives**: 
   - `alpine` → `registry.access.redhat.com/ubi8/ubi-minimal:latest`
   - `python:3.11` → `registry.access.redhat.com/ubi8/python-39:latest`
   - `node:18` → `registry.access.redhat.com/ubi8/nodejs-18:latest`

2. **Highlight Enterprise Benefits**:
   - Enhanced security with Red Hat's security patches
   - Regular vulnerability scanning and updates
   - FIPS 140-2 compliance support
   - Enterprise-grade support and SLAs
   - GPL-free licensing for commercial use

3. **Generate UBI Dockerfiles**: Automatically create UBI-compliant versions of your Dockerfiles

## Usage Examples in Claude

### Build and Validate Container Image
```
Please build a container image from https://github.com/myuser/my-app.git 
with image name quay.io/myuser/my-app:v1.0 and validate UBI compliance
```

### Build with UBI Auto-Generation
```
Build my application from the local directory ./my-app, validate for UBI compliance, 
and generate a UBI-compliant Dockerfile if needed
```

### Push to Multiple Registries
```
Push my image quay.io/myuser/my-app:latest to the registry with additional tags v1.0,stable
```

## Restart Required

**Important**: You need to restart Claude Desktop for the new configuration to take effect.

After restarting Claude Desktop, you'll have access to all the enhanced container tools with Red Hat UBI validation capabilities.

## Verification

Once Claude Desktop is restarted, you can verify the integration by asking:
```
"What container tools are available in the MCP server?"
```

The server should respond with all the new container management capabilities including UBI validation features.

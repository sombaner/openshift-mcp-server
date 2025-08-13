const vscode = require('vscode');
const https = require('https');
const http = require('http');
const path = require('path');

/**
 * OpenShift MCP Integration for Cursor/VS Code
 */

class MCPExtension {
    constructor() {
        this.requestId = 1;
        this.outputChannel = vscode.window.createOutputChannel('OpenShift MCP');
    }

    getServerUrl() {
        const config = vscode.workspace.getConfiguration('openshift-mcp');
        return config.get('serverUrl') || 'https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com';
    }

    async makeRequest(method, params = {}) {
        return new Promise((resolve, reject) => {
            const payload = JSON.stringify({
                jsonrpc: "2.0",
                method: method,
                params: params,
                id: this.requestId++
            });

            const url = new URL(this.getServerUrl());
            const options = {
                hostname: url.hostname,
                port: url.port || (url.protocol === 'https:' ? 443 : 80),
                path: url.pathname,
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Content-Length': Buffer.byteLength(payload),
                    'User-Agent': 'Cursor-MCP-Extension/1.0.0'
                },
                rejectUnauthorized: false
            };

            const client = url.protocol === 'https:' ? https : http;
            const req = client.request(options, (res) => {
                let data = '';
                res.on('data', (chunk) => {
                    data += chunk;
                });
                res.on('end', () => {
                    try {
                        const response = JSON.parse(data);
                        if (response.error) {
                            reject(new Error(response.error.message));
                        } else {
                            resolve(response.result);
                        }
                    } catch (error) {
                        reject(new Error(`Invalid JSON response: ${error.message}`));
                    }
                });
            });

            req.on('error', (error) => {
                reject(new Error(`Request failed: ${error.message}`));
            });

            req.write(payload);
            req.end();
        });
    }

    async callTool(toolName, args) {
        return this.makeRequest('tools/call', {
            name: toolName,
            arguments: args
        });
    }

    log(message) {
        this.outputChannel.appendLine(message);
        this.outputChannel.show(true);
    }

    async buildContainer() {
        try {
            // Get current workspace
            const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
            if (!workspaceFolder) {
                vscode.window.showErrorMessage('No workspace folder open');
                return;
            }

            // Ask for image name
            const imageName = await vscode.window.showInputBox({
                prompt: 'Enter container image name (e.g., quay.io/user/app:latest)',
                placeholder: 'quay.io/user/app:latest'
            });

            if (!imageName) return;

            // Show progress
            vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: 'Building Container Image',
                cancellable: false
            }, async (progress) => {
                progress.report({ message: 'Starting build with UBI validation...' });
                
                this.log(`🔨 Building container image: ${imageName}`);
                this.log(`📁 Source: ${workspaceFolder.uri.fsPath}`);

                try {
                    const result = await this.callTool('container_build', {
                        source: workspaceFolder.uri.fsPath,
                        image_name: imageName,
                        validate_ubi: true,
                        security_scan: true
                    });

                    if (result.content && result.content[0]) {
                        const buildResult = JSON.parse(result.content[0].text);
                        
                        this.log('✅ Build completed successfully!');
                        this.log(`📋 Build Summary:`);
                        this.log(`   • Image: ${buildResult.image_info?.image_name || imageName}`);
                        this.log(`   • Duration: ${buildResult.build_duration}`);
                        this.log(`   • Runtime: ${buildResult.container_runtime}`);
                        
                        if (buildResult.validation?.ubi_compliance) {
                            const ubi = buildResult.validation.ubi_compliance;
                            this.log(`   • UBI Compliant: ${ubi.is_ubi ? '✅ Yes' : '⚠️  No'}`);
                            if (!ubi.is_ubi) {
                                this.log(`   • Suggested UBI: ${ubi.suggested_ubi_image}`);
                                
                                // Offer to generate UBI Dockerfile
                                const generateUbi = await vscode.window.showInformationMessage(
                                    'Non-UBI base image detected. Generate UBI-compliant Dockerfile?',
                                    'Yes', 'No'
                                );
                                
                                if (generateUbi === 'Yes') {
                                    await this.generateUbiDockerfile(workspaceFolder.uri.fsPath, ubi.suggested_ubi_image);
                                }
                            }
                        }

                        vscode.window.showInformationMessage('Container build completed successfully!');
                    }
                } catch (error) {
                    this.log(`❌ Build failed: ${error.message}`);
                    vscode.window.showErrorMessage(`Build failed: ${error.message}`);
                }
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    }

    async generateUbiDockerfile(workspacePath, suggestedUbi) {
        try {
            // This would call the MCP server to generate a UBI Dockerfile
            // For now, we'll create a simple UBI version
            const ubiDockerfileContent = `# UBI-compliant Dockerfile generated by OpenShift MCP
FROM ${suggestedUbi}

# Copy application code
COPY . /app
WORKDIR /app

# Install dependencies and build application
# (Add your specific build steps here)

# Create non-root user for security
RUN groupadd -r appuser && useradd -r -g appuser appuser
USER appuser

# Expose port (adjust as needed)
EXPOSE 8080

# Start application
CMD ["./start.sh"]
`;

            const dockerfilePath = path.join(workspacePath, 'Dockerfile.ubi');
            const fs = require('fs');
            fs.writeFileSync(dockerfilePath, ubiDockerfileContent);
            
            this.log(`✅ Generated UBI Dockerfile: ${dockerfilePath}`);
            
            // Open the generated file
            const document = await vscode.workspace.openTextDocument(dockerfilePath);
            await vscode.window.showTextDocument(document);
            
            vscode.window.showInformationMessage('UBI Dockerfile generated successfully!');
        } catch (error) {
            this.log(`❌ Failed to generate UBI Dockerfile: ${error.message}`);
        }
    }

    async deployRepository() {
        try {
            const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
            if (!workspaceFolder) {
                vscode.window.showErrorMessage('No workspace folder open');
                return;
            }

            // Ask for Git URL and namespace
            const repoUrl = await vscode.window.showInputBox({
                prompt: 'Enter Git repository URL',
                placeholder: 'https://github.com/user/repo.git'
            });

            if (!repoUrl) return;

            const namespace = await vscode.window.showInputBox({
                prompt: 'Enter OpenShift namespace',
                placeholder: 'my-app-namespace'
            });

            if (!namespace) return;

            vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: 'Deploying Repository',
                cancellable: false
            }, async (progress) => {
                progress.report({ message: 'Starting auto-deployment...' });
                
                this.log(`🚀 Auto-deploying repository: ${repoUrl}`);
                this.log(`📦 Namespace: ${namespace}`);

                try {
                    const result = await this.callTool('repo_auto_deploy', {
                        url: repoUrl,
                        namespace: namespace
                    });

                    if (result.content && result.content[0]) {
                        const deployResult = JSON.parse(result.content[0].text);
                        
                        this.log('✅ Deployment completed!');
                        this.log(`📋 Deployment Summary:`);
                        this.log(`   • Application: ${deployResult.application_name}`);
                        this.log(`   • Namespace: ${deployResult.namespace}`);
                        if (deployResult.application_url) {
                            this.log(`   • URL: ${deployResult.application_url}`);
                        }

                        vscode.window.showInformationMessage('Repository deployed successfully!');
                    }
                } catch (error) {
                    this.log(`❌ Deployment failed: ${error.message}`);
                    vscode.window.showErrorMessage(`Deployment failed: ${error.message}`);
                }
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    }

    async listTools() {
        try {
            this.log('🛠️  Fetching available MCP tools...');
            
            const result = await this.makeRequest('tools/list');
            
            if (result.tools && Array.isArray(result.tools)) {
                this.log('\n🛠️  Available MCP Tools:\n');
                
                const toolsByCategory = this.categorizeTools(result.tools);
                
                Object.entries(toolsByCategory).forEach(([category, tools]) => {
                    this.log(`\n📂 ${category}:`);
                    tools.forEach(tool => {
                        this.log(`  • ${tool.name} - ${tool.description.substring(0, 80)}...`);
                    });
                });
                
                this.log(`\n✅ Total: ${result.tools.length} tools available`);
                vscode.window.showInformationMessage(`Found ${result.tools.length} MCP tools. Check output panel for details.`);
            } else {
                this.log('No tools found.');
                vscode.window.showWarningMessage('No MCP tools found');
            }
        } catch (error) {
            this.log(`❌ Error listing tools: ${error.message}`);
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    }

    categorizeTools(tools) {
        const categories = {
            'Container Tools': [],
            'CI/CD Tools': [],
            'Kubernetes Tools': [],
            'Other Tools': []
        };

        tools.forEach(tool => {
            if (tool.name.startsWith('container_')) {
                categories['Container Tools'].push(tool);
            } else if (tool.name.startsWith('repo_')) {
                categories['CI/CD Tools'].push(tool);
            } else if (tool.name.includes('pod') || tool.name.includes('namespace') || tool.name.includes('resource')) {
                categories['Kubernetes Tools'].push(tool);
            } else {
                categories['Other Tools'].push(tool);
            }
        });

        return categories;
    }

    async listPods() {
        try {
            const namespace = await vscode.window.showInputBox({
                prompt: 'Enter namespace (leave empty for all namespaces)',
                placeholder: 'my-namespace'
            });

            this.log(`🐳 Fetching pods${namespace ? ` in ${namespace}` : ''}...`);
            
            const method = namespace ? 'pods_list_in_namespace' : 'pods_list';
            const params = namespace ? { namespace } : {};
            
            const result = await this.callTool(method, params);

            if (result.content && result.content[0]) {
                this.log(`\n🐳 Pods${namespace ? ` in ${namespace}` : ''}:\n`);
                this.log(result.content[0].text);
                vscode.window.showInformationMessage('Pod list retrieved. Check output panel for details.');
            }
        } catch (error) {
            this.log(`❌ Error listing pods: ${error.message}`);
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    }
}

// Extension activation
function activate(context) {
    const mcpExtension = new MCPExtension();

    // Register commands
    const commands = [
        vscode.commands.registerCommand('openshift-mcp.build', () => mcpExtension.buildContainer()),
        vscode.commands.registerCommand('openshift-mcp.deploy', () => mcpExtension.deployRepository()),
        vscode.commands.registerCommand('openshift-mcp.listTools', () => mcpExtension.listTools()),
        vscode.commands.registerCommand('openshift-mcp.listPods', () => mcpExtension.listPods())
    ];

    commands.forEach(command => context.subscriptions.push(command));
    
    console.log('OpenShift MCP extension is now active!');
}

function deactivate() {}

module.exports = {
    activate,
    deactivate
};

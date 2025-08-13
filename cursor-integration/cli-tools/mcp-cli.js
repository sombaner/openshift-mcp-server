#!/usr/bin/env node

/**
 * CLI tool for interacting with OpenShift MCP Server from Cursor
 * Usage: node mcp-cli.js <command> [options]
 */

const https = require('https');
const http = require('http');

const MCP_SERVER_URL = process.env.MCP_SERVER_URL || 'https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com';

class MCPCli {
    constructor() {
        this.requestId = 1;
    }

    async makeRequest(method, params = {}) {
        return new Promise((resolve, reject) => {
            const payload = JSON.stringify({
                jsonrpc: "2.0",
                method: method,
                params: params,
                id: this.requestId++
            });

            const url = new URL(MCP_SERVER_URL);
            const options = {
                hostname: url.hostname,
                port: url.port || (url.protocol === 'https:' ? 443 : 80),
                path: url.pathname,
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Content-Length': Buffer.byteLength(payload),
                    'User-Agent': 'MCP-CLI/1.0.0'
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

    async listTools() {
        try {
            const result = await this.makeRequest('tools/list');
            console.log('\n🛠️  Available MCP Tools:\n');
            
            if (result.tools && Array.isArray(result.tools)) {
                const toolsByCategory = this.categorizeTools(result.tools);
                
                Object.entries(toolsByCategory).forEach(([category, tools]) => {
                    console.log(`\n📂 ${category}:`);
                    tools.forEach(tool => {
                        console.log(`  • ${tool.name} - ${tool.description.substring(0, 60)}...`);
                    });
                });
                
                console.log(`\n✅ Total: ${result.tools.length} tools available\n`);
            } else {
                console.log('No tools found.');
            }
        } catch (error) {
            console.error('❌ Error listing tools:', error.message);
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

    async buildContainer(source, imageName, options = {}) {
        try {
            console.log(`🔨 Building container image: ${imageName}`);
            console.log(`📁 Source: ${source}\n`);
            
            const params = {
                source: source,
                image_name: imageName,
                validate_ubi: options.validateUbi !== false,
                security_scan: options.securityScan !== false,
                ...options
            };

            const result = await this.makeRequest('tools/call', {
                name: 'container_build',
                arguments: params
            });

            if (result.content && result.content[0]) {
                const buildResult = JSON.parse(result.content[0].text);
                console.log('✅ Build completed successfully!\n');
                console.log(`📋 Build Summary:`);
                console.log(`   • Image: ${buildResult.image_info?.image_name || imageName}`);
                console.log(`   • Duration: ${buildResult.build_duration}`);
                console.log(`   • Runtime: ${buildResult.container_runtime}`);
                
                if (buildResult.validation?.ubi_compliance) {
                    const ubi = buildResult.validation.ubi_compliance;
                    console.log(`   • UBI Compliant: ${ubi.is_ubi ? '✅ Yes' : '⚠️  No'}`);
                    if (!ubi.is_ubi) {
                        console.log(`   • Suggested UBI: ${ubi.suggested_ubi_image}`);
                    }
                }
                
                console.log('\n🚀 Next steps:');
                buildResult.next_steps?.forEach(step => {
                    console.log(`   • ${step}`);
                });
            }
        } catch (error) {
            console.error('❌ Build failed:', error.message);
        }
    }

    async deployRepo(repoUrl, namespace, options = {}) {
        try {
            console.log(`🚀 Auto-deploying repository: ${repoUrl}`);
            console.log(`📦 Namespace: ${namespace}\n`);
            
            const result = await this.makeRequest('tools/call', {
                name: 'repo_auto_deploy',
                arguments: {
                    url: repoUrl,
                    namespace: namespace,
                    ...options
                }
            });

            if (result.content && result.content[0]) {
                const deployResult = JSON.parse(result.content[0].text);
                console.log('✅ Deployment completed!\n');
                console.log(`📋 Deployment Summary:`);
                console.log(`   • Application: ${deployResult.application_name}`);
                console.log(`   • Namespace: ${deployResult.namespace}`);
                if (deployResult.application_url) {
                    console.log(`   • URL: ${deployResult.application_url}`);
                }
            }
        } catch (error) {
            console.error('❌ Deployment failed:', error.message);
        }
    }

    async listPods(namespace) {
        try {
            const method = namespace ? 'pods_list_in_namespace' : 'pods_list';
            const params = namespace ? { namespace } : {};
            
            const result = await this.makeRequest('tools/call', {
                name: method,
                arguments: params
            });

            if (result.content && result.content[0]) {
                console.log(`🐳 Pods${namespace ? ` in ${namespace}` : ''}:\n`);
                console.log(result.content[0].text);
            }
        } catch (error) {
            console.error('❌ Error listing pods:', error.message);
        }
    }

    async executeWorkflow(prompt, options = {}) {
        try {
            console.log(`🤖 Analyzing prompt: "${prompt}"`);
            console.log('🔍 Determining optimal workflow...\n');
            
            const params = {
                prompt: prompt,
                dry_run: options.dryRun || false,
                interactive: options.interactive || false,
                ...options
            };

            const result = await this.makeRequest('tools/call', {
                name: 'workflow_execute',
                arguments: params
            });

            if (result.content && result.content[0]) {
                const workflowResult = JSON.parse(result.content[0].text);
                
                if (options.dryRun) {
                    console.log('🔍 Workflow Analysis (Dry Run):\n');
                    console.log(`📋 Selected Workflow: ${workflowResult.selected_workflow}`);
                    console.log(`📝 Description: ${workflowResult.workflow_description}`);
                    console.log('\n📥 Extracted Parameters:');
                    Object.entries(workflowResult.extracted_parameters).forEach(([key, value]) => {
                        console.log(`   • ${key}: ${value}`);
                    });
                    console.log('\n🔧 Steps that would be executed:');
                    workflowResult.steps_to_execute.forEach((step, index) => {
                        console.log(`   ${index + 1}. ${step.description} (${step.tool})`);
                    });
                    console.log(`\n✅ ${workflowResult.message}`);
                } else {
                    console.log('✅ Workflow Execution Completed!\n');
                    console.log(`📋 Workflow: ${workflowResult.workflow_name}`);
                    console.log(`⏱️  Duration: ${workflowResult.duration}`);
                    console.log(`🎯 Success: ${workflowResult.success ? '✅ Yes' : '❌ No'}`);
                    
                    if (workflowResult.error) {
                        console.log(`❌ Error: ${workflowResult.error}`);
                    }
                    
                    console.log('\n📊 Executed Steps:');
                    workflowResult.executed_steps.forEach((step, index) => {
                        const status = step.success ? '✅' : '❌';
                        console.log(`   ${index + 1}. ${status} ${step.tool} (${step.duration})`);
                        if (step.error) {
                            console.log(`      Error: ${step.error}`);
                        }
                    });
                    
                    if (workflowResult.recommendations && workflowResult.recommendations.length > 0) {
                        console.log('\n💡 Recommendations:');
                        workflowResult.recommendations.forEach(rec => {
                            console.log(`   ${rec}`);
                        });
                    }
                }
            }
        } catch (error) {
            console.error('❌ Workflow execution failed:', error.message);
        }
    }

    async analyzePrompt(prompt) {
        try {
            console.log(`🔍 Analyzing prompt: "${prompt}"\n`);
            
            const result = await this.makeRequest('tools/call', {
                name: 'workflow_analyze',
                arguments: {
                    prompt: prompt,
                    show_parameters: true,
                    show_confidence: true
                }
            });

            if (result.content && result.content[0]) {
                const analysis = JSON.parse(result.content[0].text);
                
                console.log('🤖 Intelligent Analysis Results:\n');
                console.log(`📋 Selected Workflow: ${analysis.selected_workflow}`);
                console.log(`📝 Description: ${analysis.workflow_description}`);
                
                console.log('\n📥 Extracted Parameters:');
                Object.entries(analysis.extracted_parameters).forEach(([key, value]) => {
                    console.log(`   • ${key}: ${value}`);
                });
                
                if (analysis.confidence_scores) {
                    console.log('\n📊 Confidence Scores:');
                    Object.entries(analysis.confidence_scores)
                        .sort((a, b) => b[1] - a[1])
                        .forEach(([workflow, score]) => {
                            const bar = '█'.repeat(Math.floor(score / 10));
                            console.log(`   ${workflow.padEnd(20)} ${score.toString().padStart(3)}% ${bar}`);
                        });
                }
                
                console.log('\n🔧 Workflow Steps:');
                analysis.workflow_steps.forEach((step, index) => {
                    console.log(`   ${index + 1}. ${step.description} (${step.tool})`);
                });
            }
        } catch (error) {
            console.error('❌ Analysis failed:', error.message);
        }
    }

    async listWorkflows() {
        try {
            const result = await this.makeRequest('tools/call', {
                name: 'workflow_list',
                arguments: {}
            });

            if (result.content && result.content[0]) {
                console.log(result.content[0].text);
            }
        } catch (error) {
            console.error('❌ Error listing workflows:', error.message);
        }
    }

    showUsage() {
        console.log(`
🛠️  OpenShift MCP CLI Tool with Intelligent Workflow Orchestration

Usage: node mcp-cli.js <command> [options]

🤖 Smart Commands:
  execute "<prompt>"              Execute workflow based on natural language prompt
  analyze "<prompt>"              Analyze prompt to show what would be executed
  workflows                       List all available workflows

📋 Traditional Commands:
  tools                           List all available MCP tools
  build <source> <image>          Build container image with UBI validation
  deploy <repo-url> <namespace>   Auto-deploy repository to OpenShift
  pods [namespace]                List pods (optionally in specific namespace)

🚀 Smart Execution Examples:
  node mcp-cli.js execute "Build and push my app from https://github.com/user/repo.git to quay.io/user/app:latest"
  node mcp-cli.js execute "Deploy container from source to production namespace"
  node mcp-cli.js execute "Scan my container image for security vulnerabilities"
  node mcp-cli.js analyze "I want to containerize my app and deploy it"

📋 Traditional Examples:
  node mcp-cli.js tools
  node mcp-cli.js build https://github.com/user/app.git quay.io/user/app:latest
  node mcp-cli.js deploy https://github.com/user/app.git my-namespace
  node mcp-cli.js pods my-namespace

Environment Variables:
  MCP_SERVER_URL    URL of the MCP server (default: OpenShift deployment)

✨ New Features:
🤖 Intelligent prompt analysis and workflow orchestration
🔗 Automatic tool chaining based on user intent
🛡️  Enhanced security scanning and UBI compliance validation
📊 Detailed execution analytics and recommendations
🚀 All operations integrate with your OpenShift CI/CD pipeline!
        `);
    }
}

// Main CLI handler
async function main() {
    const cli = new MCPCli();
    const args = process.argv.slice(2);
    
    if (args.length === 0) {
        cli.showUsage();
        return;
    }

    const command = args[0];
    
    switch (command) {
        case 'execute':
            if (args.length < 2) {
                console.error('❌ Usage: execute "<prompt>"');
                process.exit(1);
            }
            // Join all remaining args as the prompt (in case it has spaces)
            const prompt = args.slice(1).join(' ');
            await cli.executeWorkflow(prompt);
            break;
            
        case 'analyze':
            if (args.length < 2) {
                console.error('❌ Usage: analyze "<prompt>"');
                process.exit(1);
            }
            const analyzePrompt = args.slice(1).join(' ');
            await cli.analyzePrompt(analyzePrompt);
            break;
            
        case 'workflows':
            await cli.listWorkflows();
            break;
            
        case 'tools':
            await cli.listTools();
            break;
            
        case 'build':
            if (args.length < 3) {
                console.error('❌ Usage: build <source> <image>');
                process.exit(1);
            }
            await cli.buildContainer(args[1], args[2]);
            break;
            
        case 'deploy':
            if (args.length < 3) {
                console.error('❌ Usage: deploy <repo-url> <namespace>');
                process.exit(1);
            }
            await cli.deployRepo(args[1], args[2]);
            break;
            
        case 'pods':
            await cli.listPods(args[1]);
            break;
            
        default:
            console.error(`❌ Unknown command: ${command}`);
            cli.showUsage();
            process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = MCPCli;


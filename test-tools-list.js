#!/usr/bin/env node

/**
 * Test script to list all available tools from the MCP server
 */

const http = require('http');

async function listMCPTools() {
    return new Promise((resolve, reject) => {
        const postData = JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method: "tools/list"
        });

        const options = {
            hostname: 'localhost',
            port: 8080,
            path: '/mcp',
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(postData)
            }
        };

        const req = http.request(options, (res) => {
            let data = '';

            res.on('data', (chunk) => {
                data += chunk;
            });

            res.on('end', () => {
                try {
                    const response = JSON.parse(data);
                    resolve(response);
                } catch (error) {
                    reject(error);
                }
            });
        });

        req.on('error', (error) => {
            reject(error);
        });

        req.write(postData);
        req.end();
    });
}

async function main() {
    try {
        console.log('🔍 Querying MCP server for available tools...\n');
        
        const response = await listMCPTools();
        
        if (response.error) {
            console.error('❌ Error:', response.error);
            return;
        }

        if (response.result && response.result.tools) {
            const tools = response.result.tools;
            
            console.log(`✅ Found ${tools.length} tools:\n`);
            console.log('=' .repeat(80));
            
            // Group tools by category
            const categories = {
                'Container Tools': [],
                'Workflow Tools': [],
                'CI/CD Tools': [],
                'Kubernetes Tools': [],
                'Registry Tools': [],
                'Other Tools': []
            };
            
            tools.forEach(tool => {
                const name = tool.name;
                if (name.startsWith('container_')) {
                    categories['Container Tools'].push(tool);
                } else if (name.startsWith('workflow_')) {
                    categories['Workflow Tools'].push(tool);
                } else if (name.startsWith('repo_') || name.startsWith('cicd_') || name.startsWith('namespace_create')) {
                    categories['CI/CD Tools'].push(tool);
                } else if (name.startsWith('pods_') || name.startsWith('resources_') || name.startsWith('namespaces_') || name.startsWith('events_')) {
                    categories['Kubernetes Tools'].push(tool);
                } else if (name.startsWith('registry_')) {
                    categories['Registry Tools'].push(tool);
                } else {
                    categories['Other Tools'].push(tool);
                }
            });
            
            // Display tools by category
            Object.entries(categories).forEach(([category, categoryTools]) => {
                if (categoryTools.length > 0) {
                    console.log(`\n📂 ${category} (${categoryTools.length}):`);
                    console.log('-'.repeat(50));
                    categoryTools.forEach(tool => {
                        console.log(`  🔧 ${tool.name}`);
                        if (tool.description) {
                            const desc = tool.description.length > 80 
                                ? tool.description.substring(0, 80) + '...'
                                : tool.description;
                            console.log(`     ${desc}`);
                        }
                        console.log('');
                    });
                }
            });
            
            // Check for specific new tools
            console.log('\n🔍 Checking for newly implemented container tools:');
            console.log('-'.repeat(50));
            const newTools = ['container_pull', 'container_run', 'container_stop'];
            newTools.forEach(toolName => {
                const found = tools.find(t => t.name === toolName);
                if (found) {
                    console.log(`  ✅ ${toolName} - FOUND`);
                } else {
                    console.log(`  ❌ ${toolName} - NOT FOUND`);
                }
            });
            
            // Check for workflow tools
            console.log('\n🔍 Checking for workflow tools:');
            console.log('-'.repeat(50));
            const workflowTools = ['workflow_execute', 'workflow_list', 'workflow_analyze', 'workflow_create'];
            workflowTools.forEach(toolName => {
                const found = tools.find(t => t.name === toolName);
                if (found) {
                    console.log(`  ✅ ${toolName} - FOUND`);
                } else {
                    console.log(`  ❌ ${toolName} - NOT FOUND`);
                }
            });
            
        } else {
            console.log('❌ No tools found in response');
            console.log('Response:', JSON.stringify(response, null, 2));
        }
        
    } catch (error) {
        console.error('❌ Failed to query MCP server:', error.message);
        console.log('\n💡 Make sure the MCP server is running on port 8080');
        console.log('   Start with: ./openshift-mcp-server --port 8080');
    }
}

main();

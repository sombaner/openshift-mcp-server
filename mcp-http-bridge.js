#!/usr/bin/env node

/**
 * MCP HTTP Bridge for Claude Desktop Integration
 * Bridges between Claude Desktop's stdio MCP client and our HTTP-based MCP server
 */

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const https = require('https');
const http = require('http');

const MCP_SERVER_URL = process.env.MCP_SERVER_URL || 'https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com';

class MCPHttpBridge {
    constructor() {
        this.server = new Server(
            {
                name: "openshift-mcp-bridge",
                version: "1.0.0",
            },
            {
                capabilities: {
                    tools: {},
                    resources: {},
                    prompts: {},
                    logging: {}
                }
            }
        );
        this.setupHandlers();
    }

    setupHandlers() {
        // Forward tool list requests
        this.server.setRequestHandler('tools/list', async () => {
            try {
                const response = await this.makeHttpRequest('tools/list', {});
                return response.result || { tools: [] };
            } catch (error) {
                console.error('Error listing tools:', error);
                return { tools: [] };
            }
        });

        // Forward tool calls
        this.server.setRequestHandler('tools/call', async (request) => {
            try {
                const response = await this.makeHttpRequest('tools/call', request.params);
                return response.result || { content: [{ type: 'text', text: 'Error: No response from server' }] };
            } catch (error) {
                console.error('Error calling tool:', error);
                return { 
                    content: [{ 
                        type: 'text', 
                        text: `Error calling tool: ${error.message}` 
                    }] 
                };
            }
        });

        // Forward resource list requests
        this.server.setRequestHandler('resources/list', async () => {
            try {
                const response = await this.makeHttpRequest('resources/list', {});
                return response.result || { resources: [] };
            } catch (error) {
                console.error('Error listing resources:', error);
                return { resources: [] };
            }
        });

        // Forward prompt list requests
        this.server.setRequestHandler('prompts/list', async () => {
            try {
                const response = await this.makeHttpRequest('prompts/list', {});
                return response.result || { prompts: [] };
            } catch (error) {
                console.error('Error listing prompts:', error);
                return { prompts: [] };
            }
        });
    }

    makeHttpRequest(method, params) {
        return new Promise((resolve, reject) => {
            const payload = JSON.stringify({
                jsonrpc: "2.0",
                method: method,
                params: params,
                id: Date.now()
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
                    'User-Agent': 'MCP-HTTP-Bridge/1.0.0'
                },
                rejectUnauthorized: false // For self-signed certificates
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
                            reject(new Error(`Server error: ${response.error.message}`));
                        } else {
                            resolve(response);
                        }
                    } catch (error) {
                        reject(new Error(`Invalid JSON response: ${data.substring(0, 200)}...`));
                    }
                });
            });

            req.on('error', (error) => {
                reject(new Error(`HTTP request failed: ${error.message}`));
            });

            req.write(payload);
            req.end();
        });
    }

    async start() {
        const transport = new StdioServerTransport();
        await this.server.connect(transport);
        console.error('MCP HTTP Bridge started successfully');
    }
}

// Start the bridge
const bridge = new MCPHttpBridge();
bridge.start().catch((error) => {
    console.error('Failed to start MCP bridge:', error);
    process.exit(1);
});


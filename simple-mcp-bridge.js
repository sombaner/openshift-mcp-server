#!/usr/bin/env node

/**
 * Simple MCP HTTP Bridge for Claude Desktop Integration
 * A lightweight bridge between Claude Desktop and HTTP MCP server
 */

const https = require('https');
const http = require('http');

const MCP_SERVER_URL = process.env.MCP_SERVER_URL || 'https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com';

class SimpleMCPBridge {
    constructor() {
        this.requestId = 1;
        this.setupStdio();
    }

    setupStdio() {
        // Handle stdin messages from Claude Desktop
        process.stdin.setEncoding('utf8');
        let buffer = '';
        
        process.stdin.on('data', (chunk) => {
            buffer += chunk;
            this.processBuffer(buffer);
        });

        process.stdin.on('end', () => {
            process.exit(0);
        });
    }

    processBuffer(buffer) {
        // Process complete messages separated by newlines
        let remainingBuffer = buffer;
        let newlineIndex;
        
        while ((newlineIndex = remainingBuffer.indexOf('\n')) !== -1) {
            const line = remainingBuffer.substring(0, newlineIndex).trim();
            remainingBuffer = remainingBuffer.substring(newlineIndex + 1);
            
            if (line) {
                try {
                    this.handleMessage(line);
                } catch (error) {
                    console.error('Error processing line:', line, error);
                }
            }
        }
    }

    async handleMessage(messageStr) {
        let messageId = null;
        
        try {
            const message = JSON.parse(messageStr);
            messageId = message.id !== undefined ? message.id : 1;
            
            if (message.method === 'initialize') {
                // Handle initialization
                const response = {
                    jsonrpc: "2.0",
                    id: messageId,
                    result: {
                        protocolVersion: "2025-03-26",
                        capabilities: {
                            tools: {},
                            resources: {},
                            prompts: {},
                            logging: {}
                        },
                        serverInfo: {
                            name: "openshift-mcp-bridge",
                            version: "1.0.0"
                        }
                    }
                };
                this.sendResponse(response);
            } else if (message.method) {
                // Forward all other requests to HTTP server
                try {
                    const httpResponse = await this.makeHttpRequest(message.method, message.params || {});
                    
                    const response = {
                        jsonrpc: "2.0",
                        id: messageId
                    };
                    
                    // Claude Desktop expects exactly one of result or error
                    if (httpResponse && httpResponse.result !== undefined) {
                        response.result = httpResponse.result;
                    } else if (httpResponse && httpResponse.error) {
                        response.error = httpResponse.error;
                    } else {
                        // If no proper response, send error
                        response.error = {
                            code: -32603,
                            message: "No valid response from server"
                        };
                    }
                    
                    this.sendResponse(response);
                } catch (httpError) {
                    // HTTP request failed
                    this.sendResponse({
                        jsonrpc: "2.0",
                        id: messageId,
                        error: {
                            code: -32603,
                            message: "HTTP request failed",
                            data: httpError.message
                        }
                    });
                }
            } else {
                // Invalid message format
                this.sendResponse({
                    jsonrpc: "2.0",
                    id: messageId,
                    error: {
                        code: -32600,
                        message: "Invalid Request"
                    }
                });
            }
        } catch (parseError) {
            console.error('Error parsing message:', parseError);
            // Send parse error response
            this.sendResponse({
                jsonrpc: "2.0",
                id: messageId || 1,
                error: {
                    code: -32700,
                    message: "Parse error",
                    data: parseError.message
                }
            });
        }
    }

    sendResponse(response) {
        process.stdout.write(JSON.stringify(response) + '\n');
    }

    makeHttpRequest(method, params) {
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
                    'User-Agent': 'Simple-MCP-Bridge/1.0.0'
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
                        resolve(response);
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
}

// Start the bridge
const bridge = new SimpleMCPBridge();
console.error('Simple MCP Bridge started successfully');

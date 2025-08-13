#!/usr/bin/env node

/**
 * Claude Desktop MCP Proxy for OpenShift MCP Server
 * This proxy allows Claude Desktop to connect to the remote OpenShift MCP server
 */

const https = require('https');
const process = require('process');

const MCP_SERVER_URL = 'https://openshift-ai-mcp-server-mcp-ai-mcp-openshift.apps.rosa.sgaikwad.15fi.p3.openshiftapps.com';

// Forward stdin to the MCP server and stdout back to Claude
function forwardToMCPServer() {
  let buffer = '';
  
  process.stdin.on('data', (chunk) => {
    buffer += chunk.toString();
    
    // Process complete JSON-RPC messages
    const lines = buffer.split('\n');
    buffer = lines.pop() || ''; // Keep incomplete line in buffer
    
    lines.forEach(line => {
      if (line.trim()) {
        try {
          const message = JSON.parse(line);
          sendToMCPServer(message);
        } catch (err) {
          console.error('Invalid JSON:', err.message);
        }
      }
    });
  });
  
  process.stdin.on('end', () => {
    if (buffer.trim()) {
      try {
        const message = JSON.parse(buffer);
        sendToMCPServer(message);
      } catch (err) {
        console.error('Invalid JSON in buffer:', err.message);
      }
    }
  });
}

function sendToMCPServer(message) {
  const postData = JSON.stringify(message);
  
  const options = {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Content-Length': Buffer.byteLength(postData)
    }
  };
  
  const req = https.request(MCP_SERVER_URL, options, (res) => {
    let responseData = '';
    
    res.on('data', (chunk) => {
      responseData += chunk;
    });
    
    res.on('end', () => {
      try {
        const response = JSON.parse(responseData);
        process.stdout.write(JSON.stringify(response) + '\n');
      } catch (err) {
        console.error('Invalid response JSON:', err.message);
      }
    });
  });
  
  req.on('error', (err) => {
    console.error('Request error:', err.message);
    process.exit(1);
  });
  
  req.write(postData);
  req.end();
}

// Start the proxy
forwardToMCPServer();

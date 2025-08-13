#!/usr/bin/env node

// Quick test of the bridge functionality
const { spawn } = require('child_process');

async function testBridge() {
    console.log('Testing MCP Bridge...');
    
    const bridge = spawn('node', ['/Users/sureshgaikwad/openshift-mcp-server/simple-mcp-bridge.js'], {
        stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let output = '';
    bridge.stdout.on('data', (data) => {
        output += data.toString();
        console.log('Bridge Response:', data.toString().trim());
    });
    
    bridge.stderr.on('data', (data) => {
        console.log('Bridge Error:', data.toString().trim());
    });
    
    // Test initialization
    console.log('\n1. Testing initialization...');
    bridge.stdin.write('{"jsonrpc": "2.0", "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}, "id": 1}\n');
    
    // Wait a bit then test tools list
    setTimeout(() => {
        console.log('\n2. Testing tools list...');
        bridge.stdin.write('{"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": 2}\n');
        
        // Close after another delay
        setTimeout(() => {
            bridge.kill();
            process.exit(0);
        }, 3000);
    }, 2000);
}

testBridge();



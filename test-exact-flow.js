#!/usr/bin/env node

// Test the exact flow that Claude Desktop uses
const { spawn } = require('child_process');

async function testExactFlow() {
    console.log('Testing exact Claude Desktop flow...');
    
    const bridge = spawn('node', ['/Users/sureshgaikwad/openshift-mcp-server/simple-mcp-bridge.js'], {
        stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let responses = [];
    bridge.stdout.on('data', (data) => {
        const output = data.toString().trim();
        console.log('Bridge Response:', output);
        responses.push(output);
    });
    
    bridge.stderr.on('data', (data) => {
        console.log('Bridge Error:', data.toString().trim());
    });
    
    // Send exact initialization from Claude Desktop logs
    console.log('\n1. Sending Claude Desktop initialization (id: 0)...');
    bridge.stdin.write('{"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-ai","version":"0.1.0"}},"jsonrpc":"2.0","id":0}\n');
    
    // Wait then check if we got response with id: 0
    setTimeout(() => {
        console.log('\n2. Checking responses...');
        const initResponse = responses.find(r => r.includes('"id":0'));
        if (initResponse) {
            console.log('✅ Found correct initialization response with id: 0');
            
            // Now test tools list
            console.log('\n3. Testing tools/list...');
            bridge.stdin.write('{"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": 1}\n');
            
            setTimeout(() => {
                bridge.kill();
                process.exit(0);
            }, 3000);
        } else {
            console.log('❌ No response found with id: 0');
            console.log('Available responses:', responses);
            bridge.kill();
            process.exit(1);
        }
    }, 2000);
}

testExactFlow();

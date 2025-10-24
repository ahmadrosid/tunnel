#!/usr/bin/env node

/**
 * Tunnel Client - WebSocket-based tunneling client
 *
 * Usage:
 *   node client.js [subdomain] [local-port]
 *
 * Examples:
 *   node client.js myapp 3000           # Custom subdomain
 *   node client.js - 3000                # Random subdomain
 *
 * Environment variables (or use .env file):
 *   TUNNEL_SERVER - Tunnel server URL (default: ws://localhost:8080/tunnel)
 *   SUBDOMAIN     - Default subdomain (can be overridden by CLI arg)
 *   LOCAL_PORT    - Default local port (can be overridden by CLI arg)
 *   LOCAL_HOST    - Local host (default: localhost)
 */

// Load environment variables from .env file
require('dotenv').config();

const WebSocket = require('ws');
const http = require('http');

// Configuration - prioritize CLI args, then env vars, then defaults
const TUNNEL_SERVER = process.env.TUNNEL_SERVER || 'ws://localhost:8080/tunnel';
const subdomain = process.argv[2] === '-' ? '' : (process.argv[2] || process.env.SUBDOMAIN || '');
const localPort = parseInt(process.argv[3] || process.env.LOCAL_PORT || '3000');
const localHost = process.env.LOCAL_HOST || 'localhost';

console.log('🚀 Starting tunnel client...');
console.log(`📍 Server: ${TUNNEL_SERVER}`);
console.log(`🏠 Local: http://${localHost}:${localPort}`);
console.log(`🏷️  Subdomain: ${subdomain || '(random)'}\n`);

// Connect to tunnel server
const ws = new WebSocket(TUNNEL_SERVER);

ws.on('open', () => {
  console.log('✅ Connected to tunnel server');

  // Register tunnel
  const registerMsg = {
    type: 'register',
    timestamp: new Date().toISOString(),
    data: {
      subdomain: subdomain,
      local_addr: `${localHost}:${localPort}`,
      local_port: localPort
    }
  };

  ws.send(JSON.stringify(registerMsg));
  console.log('📤 Sent registration request...');
});

ws.on('message', (data) => {
  try {
    const msg = JSON.parse(data.toString());

    if (msg.type === 'success') {
      const info = msg.data;
      console.log('\n✨ Tunnel created successfully!');
      console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
      console.log(`🌐 Public URL: https://${info.full_domain}`);
      console.log(`📌 Subdomain: ${info.subdomain}`);
      console.log(`🔗 Forwarding to: ${info.local_addr}`);
      console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`);
      console.log('💡 Press Ctrl+C to stop the tunnel');
    } else if (msg.type === 'error') {
      console.error(`❌ Error: ${msg.error}`);
      process.exit(1);
    } else if (msg.type === 'pong') {
      // Ignore pong messages
    } else {
      console.log(`📨 Received: ${msg.type}`);
    }
  } catch (err) {
    // Binary data - this is HTTP traffic to forward
    handleHttpTraffic(data);
  }
});

ws.on('error', (err) => {
  console.error(`❌ WebSocket error: ${err.message}`);
});

ws.on('close', () => {
  console.log('👋 Disconnected from tunnel server');
  process.exit(0);
});

// Handle incoming HTTP traffic from tunnel
function handleHttpTraffic(data) {
  // Forward the HTTP request to local server
  const options = {
    hostname: localHost,
    port: localPort,
    method: 'GET',
    path: '/',
    headers: {}
  };

  // Parse HTTP request (simplified - would need proper HTTP parsing in production)
  const request = http.request(options, (response) => {
    let responseData = '';

    response.on('data', (chunk) => {
      responseData += chunk;
    });

    response.on('end', () => {
      // Build HTTP response
      const httpResponse = `HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\n`;
      const headers = Object.entries(response.headers)
        .map(([key, value]) => `${key}: ${value}`)
        .join('\r\n');
      const fullResponse = `${httpResponse}${headers}\r\n\r\n${responseData}`;

      // Send response back through tunnel
      ws.send(Buffer.from(fullResponse), { binary: true });
    });
  });

  request.on('error', (err) => {
    console.error(`❌ Local server error: ${err.message}`);

    // Send 502 Bad Gateway response
    const errorResponse = 'HTTP/1.1 502 Bad Gateway\r\n' +
                         'Content-Type: text/plain\r\n' +
                         'Content-Length: 15\r\n\r\n' +
                         'Bad Gateway\r\n';
    ws.send(Buffer.from(errorResponse), { binary: true });
  });

  // Forward the request
  request.write(data);
  request.end();
}

// Send ping every 30 seconds to keep connection alive
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'ping',
      timestamp: new Date().toISOString()
    }));
  }
}, 30000);

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\n\n🛑 Shutting down...');
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'unregister',
      timestamp: new Date().toISOString()
    }));
    ws.close();
  }
  setTimeout(() => process.exit(0), 1000);
});

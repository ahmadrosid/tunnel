#!/usr/bin/env node

// Quick test to see actual errors
require('dotenv').config();
const WebSocket = require('ws');

const TUNNEL_SERVER = process.env.TUNNEL_SERVER || 'wss://easypod.cloud/tunnel';
const subdomain = process.argv[2] || 'test123';
const localPort = parseInt(process.argv[3] || '3000');

console.log('🚀 Testing tunnel client...');
console.log(`📍 Server: ${TUNNEL_SERVER}`);
console.log(`🏠 Local: http://localhost:${localPort}`);
console.log(`🏷️  Subdomain: ${subdomain}\n`);

const ws = new WebSocket(TUNNEL_SERVER);

ws.on('open', () => {
  console.log('✅ Connected');
  ws.send(JSON.stringify({
    type: 'register',
    timestamp: new Date().toISOString(),
    data: {
      subdomain: subdomain,
      local_addr: `localhost:${localPort}`,
      local_port: localPort
    }
  }));
});

ws.on('message', (data, isBinary) => {
  console.log(`📨 Message received (binary: ${isBinary}, size: ${data.length} bytes)`);
  
  if (isBinary) {
    console.log('   Binary data:', data.toString('utf8').substring(0, 100));
  } else {
    try {
      const msg = JSON.parse(data.toString());
      console.log('   JSON:', msg);
    } catch (e) {
      console.log('   Text:', data.toString().substring(0, 100));
    }
  }
});

ws.on('error', (err) => {
  console.error('❌ Error:', err.message);
});

ws.on('close', (code, reason) => {
  console.log(`👋 Closed (code: ${code}, reason: ${reason})`);
  process.exit(0);
});

// Keep alive
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    console.log('🏓 Sending ping...');
    ws.send(JSON.stringify({ type: 'ping', timestamp: new Date().toISOString() }));
  }
}, 30000);

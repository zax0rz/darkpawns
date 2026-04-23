---
title: "Dark Pawns Documentation"
description: "Documentation for Dark Pawns MUD - A resurrection of the 1997-2010 MUD with AI agents as first-class players"
date: 2026-04-22
agent_friendly: true
copy_paste_commands:
  - id: "connect-python"
    language: "python"
    code: |
      import websocket
      import json
      
      # Connect to Dark Pawns WebSocket
      ws = websocket.WebSocket()
      ws.connect("ws://localhost:4350/ws")
      
      # Login as agent
      login_msg = {
          "type": "login",
          "data": {
              "player_name": "test-agent",
              "api_key": "your-api-key-here",
              "mode": "agent"
          }
      }
      ws.send(json.dumps(login_msg))
      response = json.loads(ws.recv())
      print("Login response:", response)
      
      # Send a command
      command_msg = {
          "type": "command",
          "data": {
              "command": "look"
          }
      }
      ws.send(json.dumps(command_msg))
      response = json.loads(ws.recv())
      print("Command response:", response)
  - id: "connect-node"
    language: "javascript"
    code: |
      const WebSocket = require('ws');
      
      // Connect to Dark Pawns WebSocket
      const ws = new WebSocket('ws://localhost:4350/ws');
      
      ws.on('open', function open() {
          // Login as agent
          const loginMsg = {
              type: 'login',
              data: {
                  player_name: 'test-agent',
                  api_key: 'your-api-key-here',
                  mode: 'agent'
              }
          };
          ws.send(JSON.stringify(loginMsg));
      });
      
      ws.on('message', function message(data) {
          const response = JSON.parse(data);
          console.log('Received:', response);
          
          // Send look command after login
          if (response.type === 'login_response' && response.data.success) {
              const commandMsg = {
                  type: 'command',
                  data: {
                      command: 'look'
                  }
              };
              ws.send(JSON.stringify(commandMsg));
          }
      });
api_examples:
  - title: "WebSocket RPC Methods"
    description: "Typed WebSocket methods with error handling"
    language: "python"
    code: |
      class DarkPawnsClient:
          def __init__(self, url="ws://localhost:4350/ws"):
              self.ws = websocket.WebSocket()
              self.ws.connect(url)
              self.connected = False
          
          def login(self, player_name, api_key=None, mode="agent"):
              """Login to Dark Pawns server"""
              msg = {
                  "type": "login",
                  "data": {
                      "player_name": player_name,
                      "api_key": api_key,
                      "mode": mode
                  }
              }
              self.ws.send(json.dumps(msg))
              response = json.loads(self.ws.recv())
              
              if response.get("type") == "error":
                  raise Exception(f"Login failed: {response['data']['message']}")
              
              self.connected = True
              return response
          
          def command(self, cmd, args=None):
              """Send a game command"""
              if not self.connected:
                  raise Exception("Not connected. Call login() first.")
              
              msg = {
                  "type": "command",
                  "data": {
                      "command": cmd,
                      "args": args or []
                  }
              }
              self.ws.send(json.dumps(msg))
              return json.loads(self.ws.recv())
          
          def subscribe(self, variables):
              """Subscribe to game state variables"""
              msg = {
                  "type": "subscribe",
                  "data": {
                      "variables": variables
                  }
              }
              self.ws.send(json.dumps(msg))
              return json.loads(self.ws.recv())
          
          def close(self):
              """Close the connection"""
              self.ws.close()
              self.connected = False
---

# Welcome to Dark Pawns Documentation

Dark Pawns is a resurrection of the Dark Pawns MUD that ran from 1997 to 2010. This documentation covers everything you need to know about the game, from playing as a human to integrating AI agents as first-class players.

## What's Different About This Documentation?

This documentation site is built with **dual rendering** in mind:

- **For Humans**: Beautiful HTML pages with navigation, examples, and explanations
- **For Agents**: Markdown versions accessible via `Accept: text/markdown` header
- **Structured Data**: OpenAPI specifications, JSON-LD, and machine-readable content
- **Copy/Paste Ready**: Code examples you can use immediately

## Quick Links

### For Players
- [Getting Started](/getting-started/) - How to connect and start playing
- [Game Commands](/game/commands/) - Complete command reference
- [World Guide](/game/world/) - Explore the Dark Pawns world

### For Agent Developers
- [Agent Integration Guide](/agents/) - Connect AI agents to Dark Pawns
- [WebSocket Protocol](/agents/protocol/) - Complete protocol specification
- [Example Agents](/agents/examples/) - Reference implementations

### For Contributors
- [API Reference](/api/) - Complete API documentation
- [Development Guide](/development/) - How to contribute to the project
- [Architecture](/development/architecture/) - System design and components

## Content Negotiation

Agents can access markdown versions of any page by setting the `Accept: text/markdown` header:

```bash
# Get HTML (default)
curl https://darkpawns.labz0rz.com/docs/

# Get Markdown for agents
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/docs/

# Get OpenAPI specification
curl https://darkpawns.labz0rz.com/docs/api/openapi.json
```

## Search Functionality

This documentation includes full-text search that works for both humans and agents. The search index is available at `/docs/search-index.json` and includes:

- Page titles and descriptions
- Full content (for relevant pages)
- Tags and categories
- URLs for direct access

## Agent-Friendly Features

1. **Structured Data**: All pages include metadata for machine parsing
2. **Code Examples**: Ready-to-use examples in Python, JavaScript, and Go
3. **API Specifications**: Complete OpenAPI 3.0 specification
4. **WebSocket Examples**: Typed RPC methods with error handling
5. **Rate Limit Information**: Clear documentation of API limits

## Getting Help

- **Discord**: Join our [Discord community](https://discord.gg/darkpawns)
- **GitHub**: Report issues on [GitHub](https://github.com/zax0rz/darkpawns/issues)
- **Email**: Contact us at hello@labz0rz.com

---

*Dark Pawns was originally created by the Dark Pawns Coding Team (1997–2010). This is a faithful resurrection with modern infrastructure and AI agent support.*
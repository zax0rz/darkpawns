---
title: "WebSocket Protocol"
description: "Complete WebSocket protocol specification for Dark Pawns agents"
date: 2026-04-22
agent_friendly: true
copy_paste_commands:
  - id: "agent-login"
    language: "python"
    code: |
      import websocket
      import json
      import time
      
      class DarkPawnsAgent:
          def __init__(self, url="ws://localhost:4350/ws"):
              self.ws = websocket.WebSocket()
              self.ws.connect(url)
              self.player_name = None
              self.api_key = None
              self.state = {}
              self.subscriptions = []
          
          def login(self, player_name, api_key):
              """Login to Dark Pawns as an agent"""
              self.player_name = player_name
              self.api_key = api_key
              
              msg = {
                  "type": "login",
                  "data": {
                      "player_name": player_name,
                      "api_key": api_key,
                      "mode": "agent"
                  }
              }
              self.ws.send(json.dumps(msg))
              response = json.loads(self.ws.recv())
              
              if response.get("type") == "error":
                  raise Exception(f"Login failed: {response['data']['message']}")
              
              print(f"Logged in as {player_name}")
              return response
          
          def subscribe(self, variables):
              """Subscribe to game state variables"""
              self.subscriptions = variables
              
              msg = {
                  "type": "subscribe",
                  "data": {
                      "variables": variables
                  }
              }
              self.ws.send(json.dumps(msg))
              response = json.loads(self.ws.recv())
              
              if response.get("type") == "error":
                  print(f"Subscription failed: {response['data']['message']}")
              else:
                  print(f"Subscribed to: {', '.join(variables)}")
              
              return response
          
          def send_command(self, command, args=None):
              """Send a game command"""
              msg = {
                  "type": "command",
                  "data": {
                      "command": command,
                      "args": args or []
                  }
              }
              self.ws.send(json.dumps(msg))
              return json.loads(self.ws.recv())
          
          def poll_messages(self, timeout=1):
              """Poll for incoming messages"""
              self.ws.settimeout(timeout)
              try:
                  data = self.ws.recv()
                  if data:
                      message = json.loads(data)
                      self._handle_message(message)
                      return message
              except websocket.WebSocketTimeoutException:
                  pass
              return None
          
          def _handle_message(self, message):
              """Handle incoming messages"""
              msg_type = message.get("type")
              
              if msg_type == "state_update":
                  # Update local state
                  self.state.update(message["data"])
                  print(f"State updated: {list(message['data'].keys())}")
              
              elif msg_type == "command_response":
                  print(f"Command response: {message['data']['message']}")
              
              elif msg_type == "error":
                  print(f"Error: {message['data']['message']}")
          
          def run_loop(self):
              """Main agent loop"""
              print("Starting agent loop...")
              try:
                  while True:
                      # Poll for messages
                      self.poll_messages()
                      
                      # Agent logic here
                      # Example: Move randomly if not in combat
                      if not self.state.get("FIGHTING"):
                          exits = self.state.get("ROOM_EXITS", [])
                          if exits:
                              import random
                              direction = random.choice(exits)
                              self.send_command(direction)
                      
                      time.sleep(0.5)
              except KeyboardInterrupt:
                  print("Agent stopped")
              finally:
                  self.ws.close()
      
      # Usage
      if __name__ == "__main__":
          agent = DarkPawnsAgent()
          agent.login("brenda69", "your-api-key-here")
          agent.subscribe(["HEALTH", "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS", "FIGHTING"])
          agent.run_loop()
api_examples:
  - title: "Complete Agent Implementation"
    description: "Full-featured agent with state management and decision making"
    language: "python"
    code: |
      import websocket
      import json
      import time
      from typing import Dict, List, Optional
      from dataclasses import dataclass, field
      from enum import Enum
      
      class AgentState(Enum):
          IDLE = "idle"
          EXPLORING = "exploring"
          COMBAT = "combat"
          LOOTING = "looting"
          RESTING = "resting"
      
      @dataclass
      class GameState:
          health: int = 100
          max_health: int = 100
          mana: int = 100
          level: int = 1
          room_vnum: Optional[int] = None
          room_name: Optional[str] = None
          room_exits: List[str] = field(default_factory=list)
          room_mobs: List[Dict] = field(default_factory=list)
          room_items: List[Dict] = field(default_factory=list)
          fighting: Optional[Dict] = None
          inventory: List[Dict] = field(default_factory=list)
          equipment: Dict = field(default_factory=dict)
      
      class DarkPawnsAgent:
          def __init__(self, url: str = "ws://localhost:4350/ws"):
              self.ws = websocket.WebSocket()
              self.ws.connect(url)
              self.state = GameState()
              self.agent_state = AgentState.IDLE
              self.memory = []  # Simple memory for past actions
              
          def connect(self, player_name: str, api_key: str) -> bool:
              """Establish connection and login"""
              login_msg = {
                  "type": "login",
                  "data": {
                      "player_name": player_name,
                      "api_key": api_key,
                      "mode": "agent"
                  }
              }
              self.ws.send(json.dumps(login_msg))
              response = json.loads(self.ws.recv())
              
              if response.get("type") == "error":
                  print(f"Login failed: {response['data']['message']}")
                  return False
              
              print(f"Connected as {player_name}")
              return True
          
          def subscribe_to_all(self):
              """Subscribe to all available state variables"""
              variables = [
                  "HEALTH", "MAX_HEALTH", "MANA", "LEVEL",
                  "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS",
                  "ROOM_MOBS", "ROOM_ITEMS", "FIGHTING",
                  "INVENTORY", "EQUIPMENT", "EVENTS"
              ]
              
              subscribe_msg = {
                  "type": "subscribe",
                  "data": {"variables": variables}
              }
              self.ws.send(json.dumps(subscribe_msg))
              response = json.loads(self.ws.recv())
              
              if response.get("type") == "error":
                  print(f"Subscription error: {response['data']['message']}")
              else:
                  print("Subscribed to all state variables")
              
              return response
          
          def update_state(self, update: Dict):
              """Update local game state from server message"""
              for key, value in update.items():
                  if hasattr(self.state, key.lower()):
                      setattr(self.state, key.lower(), value)
              
              # Update agent state based on game state
              if self.state.fighting:
                  self.agent_state = AgentState.COMBAT
              elif self.state.health < self.state.max_health * 0.3:
                  self.agent_state = AgentState.RESTING
              elif self.state.room_items:
                  self.agent_state = AgentState.LOOTING
              else:
                  self.agent_state = AgentState.EXPLORING
          
          def decide_action(self) -> Optional[Dict]:
              """Decide next action based on current state"""
              if self.agent_state == AgentState.COMBAT:
                  return self._combat_action()
              elif self.agent_state == AgentState.RESTING:
                  return self._rest_action()
              elif self.agent_state == AgentState.LOOTING:
                  return self._loot_action()
              elif self.agent_state == AgentState.EXPLORING:
                  return self._explore_action()
              return None
          
          def _combat_action(self) -> Dict:
              """Actions to take during combat"""
              if self.state.fighting:
                  return {
                      "type": "command",
                      "data": {
                          "command": "hit",
                          "args": [self.state.fighting.get("name", "target")]
                      }
                  }
              return {"type": "command", "data": {"command": "flee"}}
          
          def _rest_action(self) -> Dict:
              """Actions to take when resting"""
              if self.state.health < self.state.max_health * 0.5:
                  return {"type": "command", "data": {"command": "rest"}}
              return self._explore_action()
          
          def _loot_action(self) -> Dict:
              """Actions to take when looting"""
              if self.state.room_items:
                  item = self.state.room_items[0]
                  return {
                      "type": "command",
                      "data": {
                          "command": "get",
                          "args": [item.get("name", "item")]
                      }
                  }
              return self._explore_action()
          
          def _explore_action(self) -> Dict:
              """Actions to take when exploring"""
              if self.state.room_exits:
                  # Choose an exit we haven't taken recently
                  recent_exits = [m.get("exit") for m in self.memory[-5:] if m.get("type") == "move"]
                  available_exits = [e for e in self.state.room_exits if e not in recent_exits]
                  
                  if available_exits:
                      direction = available_exits[0]
                  else:
                      direction = self.state.room_exits[0]
                  
                  self.memory.append({"type": "move", "exit": direction, "room": self.state.room_vnum})
                  return {"type": "command", "data": {"command": direction}}
              
              return {"type": "command", "data": {"command": "look"}}
          
          def run(self, interval: float = 1.0):
              """Main agent execution loop"""
              print("Starting agent...")
              try:
                  while True:
                      # Check for incoming messages
                      self.ws.settimeout(0.1)
                      try:
                          data = self.ws.recv()
                          if data:
                              message = json.loads(data)
                              if message.get("type") == "state_update":
                                  self.update_state(message["data"])
                              print(f"Received: {message.get('type')}")
                      except websocket.WebSocketTimeoutException:
                          pass
                      
                      # Decide and execute action
                      action = self.decide_action()
                      if action:
                          self.ws.send(json.dumps(action))
                          print(f"Sent: {action['data']['command']}")
                      
                      time.sleep(interval)
              except KeyboardInterrupt:
                  print("Agent stopped")
              finally:
                  self.ws.close()
      
      # Example usage
      if __name__ == "__main__":
          agent = DarkPawnsAgent()
          if agent.connect("brenda69", "your-api-key-here"):
              agent.subscribe_to_all()
              agent.run()
---

# WebSocket Protocol Specification

Dark Pawns uses a WebSocket-based protocol for real-time communication between clients and the game server. The protocol supports both human players (text mode) and AI agents (JSON mode).

## Connection Details

- **URL**: `ws://localhost:4350/ws` (development) or `wss://darkpawns.labz0rz.com/ws` (production)
- **Protocol**: WebSocket (RFC 6455)
- **Message Format**: JSON for agents, plain text for humans
- **Rate Limit**: 10 commands per second per connection

## Message Types

### 1. Login (`type: "login"`)

Required for all connections. Determines whether the client is a human player or an AI agent.

**Request:**
```json
{
  "type": "login",
  "data": {
    "player_name": "brenda69",
    "api_key": "agent-key-123",
    "mode": "agent"
  }
}
```

**Response (success):**
```json
{
  "type": "login_response",
  "data": {
    "success": true,
    "message": "Welcome to Dark Pawns, brenda69!"
  }
}
```

**Response (error):**
```json
{
  "type": "error",
  "data": {
    "code": "AUTH_FAILED",
    "message": "Invalid API key"
  }
}
```

### 2. Command (`type: "command"`)

Send game commands to the server.

**Request:**
```json
{
  "type": "command",
  "data": {
    "command": "look",
    "args": []
  }
}
```

**Response:**
```json
{
  "type": "command_response",
  "data": {
    "success": true,
    "message": "You look around...",
    "output": "The Town Square\nYou are in the bustling town square. Exits: north, east, south, west."
  }
}
```

### 3. Subscription (`type: "subscribe"`)

Agents can subscribe to game state variables to receive automatic updates.

**Request:**
```json
{
  "type": "subscribe",
  "data": {
    "variables": ["HEALTH", "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS"]
  }
}
```

**Response:**
```json
{
  "type": "subscription_response",
  "data": {
    "success": true,
    "subscribed": ["HEALTH", "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS"]
  }
}
```

### 4. State Update (`type: "state_update"`)

Server pushes state updates to subscribed agents when variables change.

**Message:**
```json
{
  "type": "state_update",
  "data": {
    "HEALTH": 85,
    "MAX_HEALTH": 100,
    "ROOM_VNUM": 3001,
    "ROOM_NAME": "The Town Square",
    "ROOM_EXITS": ["north", "east", "south", "west"],
    "ROOM_MOBS": [
      {
        "vnum": 1001,
        "name": "a town guard",
        "short_desc": "A vigilant town guard stands here.",
        "health": 100
      }
    ]
  }
}
```

## Available State Variables

Agents can subscribe to these variables:

| Variable | Type | Description |
|----------|------|-------------|
| `HEALTH` | integer | Current health points |
| `MAX_HEALTH` | integer | Maximum health points |
| `MANA` | integer | Current mana points |
| `LEVEL` | integer | Character level |
| `ROOM_VNUM` | integer | Current room virtual number |
| `ROOM_NAME` | string | Current room name |
| `ROOM_EXITS` | array | Available exits from current room |
| `ROOM_MOBS` | array | Mobs in current room (with `target_string` for combat) |
| `ROOM_ITEMS` | array | Items in current room |
| `FIGHTING` | object | Current combat target (null if not fighting) |
| `INVENTORY` | array | Items in inventory |
| `EQUIPMENT` | object | Equipped items by slot |
| `EVENTS` | array | Recent game events |

## Rate Limiting

- **Command Rate**: 10 commands per second (token bucket algorithm)
- **Connection Limit**: 5 concurrent connections per IP
- **Message Size**: 16KB maximum per message

Rate limit headers are included in error responses:
```json
{
  "type": "error",
  "data": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded",
    "retry_after": 1.5
  }
}
```

## Error Handling

All error responses follow this format:
```json
{
  "type": "error",
  "data": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}  // Optional additional details
  }
}
```

Common error codes:
- `AUTH_FAILED`: Invalid API key or authentication failure
- `RATE_LIMITED`: Rate limit exceeded
- `INVALID_COMMAND`: Unknown or malformed command
- `PLAYER_NOT_FOUND`: Player doesn't exist
- `INTERNAL_ERROR`: Server error

## Best Practices for Agents

1. **Always subscribe to state variables** you need rather than polling with commands
2. **Handle rate limiting** with exponential backoff
3. **Maintain connection state** and reconnect on failure
4. **Use the `target_string` field** from `ROOM_MOBS` for combat commands
5. **Implement health-based decision making** (rest when low health)
6. **Log all actions** for debugging and learning

## Example Agent State Machine

```python
# Pseudo-code for agent decision making
def decide_action(state):
    if state.fighting:
        return "hit " + state.fighting.target_string
    elif state.health < state.max_health * 0.3:
        return "rest"
    elif state.room_items:
        return "get " + state.room_items[0].name
    elif state.room_exits:
        return choose_exit(state.room_exits)
    else:
        return "look"
```

## Testing Your Agent

Use the provided test script:
```bash
# Test agent connection
python3 -m scripts.test_agent --name test-agent --key your-api-key

# Test with specific commands
python3 -m scripts.test_agent --command "look;north;east"

# Test rate limiting
python3 -m scripts.test_agent --stress-test
```

## Next Steps

- Read the [Example Agents](/agents/examples/) page for complete implementations
- Check the [API Reference](/api/) for detailed endpoint documentation
- Join the [Discord community](https://discord.gg/darkpawns) for help and discussion
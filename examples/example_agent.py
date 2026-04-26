import websocket
import json
import time
import threading

class DarkPawnsAgent:
    def __init__(self, api_key, player_name="agent"):
        self.ws = websocket.WebSocket()
        self.api_key = api_key
        self.player_name = player_name
        self.state = {}
        self.variables = {}
        self.running = True
        
    def connect(self, url="ws://localhost:4350/ws"):
        """Connect, login, and subscribe to variables"""
        self.ws.connect(url)
        
        # Login
        login_msg = {
            "type": "login",
            "data": {
                "player_name": self.player_name,
                "api_key": self.api_key,
                "mode": "agent"
            }
        }
        self.ws.send(json.dumps(login_msg))
        
        # Receive initial state
        response = json.loads(self.ws.recv())
        if response.get("type") == "state":
            self.state = response.get("data", {})
            print(f"Logged in as {self.state.get('player', {}).get('name')}")
        
        # Subscribe to all variables
        subscribe_msg = {
            "type": "subscribe",
            "data": {
                "variables": [
                    "HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA",
                    "LEVEL", "EXP", "ROOM_VNUM", "ROOM_NAME",
                    "ROOM_EXITS", "ROOM_MOBS", "ROOM_ITEMS",
                    "FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS"
                ]
            }
        }
        self.ws.send(json.dumps(subscribe_msg))
        print("Subscribed to all variables")
        
        # Start message listener thread
        self.listener_thread = threading.Thread(target=self._message_listener)
        self.listener_thread.daemon = True
        self.listener_thread.start()
        
        return response
    
    def _message_listener(self):
        """Background thread to handle incoming messages"""
        while self.running:
            try:
                response = json.loads(self.ws.recv())
                msg_type = response.get("type")
                data = response.get("data", {})
                
                if msg_type == "vars":
                    # Update variables from vars message
                    self.variables.update(data)
                    # Also update state for backward compatibility
                    self._update_state_from_vars(data)
                elif msg_type == "state":
                    self.state = data
                elif msg_type == "event":
                    print(f"Event: {data.get('text')}")
                elif msg_type == "error":
                    print(f"Error: {data.get('message')}")
                elif msg_type == "text":
                    print(f"Text: {data.get('text')}")
                
            except Exception as e:
                if self.running:
                    print(f"Message listener error: {e}")
    
    def _update_state_from_vars(self, vars_data):
        """Update state object from variable updates"""
        if "HEALTH" in vars_data:
            if "player" not in self.state:
                self.state["player"] = {}
            self.state["player"]["health"] = vars_data["HEALTH"]
        if "MAX_HEALTH" in vars_data:
            if "player" not in self.state:
                self.state["player"] = {}
            self.state["player"]["max_health"] = vars_data["MAX_HEALTH"]
        if "ROOM_VNUM" in vars_data:
            if "room" not in self.state:
                self.state["room"] = {}
            self.state["room"]["vnum"] = vars_data["ROOM_VNUM"]
        if "ROOM_NAME" in vars_data:
            if "room" not in self.state:
                self.state["room"] = {}
            self.state["room"]["name"] = vars_data["ROOM_NAME"]
        
    def command(self, cmd, args=None):
        """Send a command"""
        msg = {
            "type": "command",
            "data": {"command": cmd}
        }
        if args:
            msg["data"]["args"] = args
        
        self.ws.send(json.dumps(msg))
        print(f"Command: {cmd} {args if args else ''}")
        
        # Commands don't get immediate responses - state updates come via vars messages
        # Wait a bit for vars to update
        time.sleep(0.1)
        
    def get_health(self):
        """Get current health from variables"""
        return self.variables.get("HEALTH", 100)
    
    def get_room_mobs(self):
        """Get mobs in current room"""
        return self.variables.get("ROOM_MOBS", [])
    
    def explore(self):
        """Basic exploration behavior"""
        print("Starting exploration...")
        
        # Look around
        self.command("look")
        time.sleep(0.5)
        
        # Check room info
        room_name = self.variables.get("ROOM_NAME", "Unknown")
        room_exits = self.variables.get("ROOM_EXITS", [])
        room_mobs = self.variables.get("ROOM_MOBS", [])
        
        print(f"Room: {room_name}")
        print(f"Exits: {', '.join(room_exits)}")
        
        if room_mobs:
            print(f"Found mobs: {[m['name'] for m in room_mobs]}")
            # Attack first mob
            if room_mobs:
                target = room_mobs[0]['target_string']
                print(f"Attacking {target}...")
                self.command("hit", [target])
                
                # Simple combat loop
                for _ in range(5):  # Limit to 5 rounds
                    time.sleep(2.1)  # Combat tick rate
                    health = self.get_health()
                    print(f"Health: {health}")
                    if health < 30:
                        print("Health low - fleeing!")
                        self.command("flee")
                        break
        
        # Check for items
        room_items = self.variables.get("ROOM_ITEMS", [])
        if room_items:
            print(f"Found items: {[i['name'] for i in room_items]}")
            # Pick up first item
            if room_items:
                item = room_items[0]['target_string']
                print(f"Getting {item}...")
                self.command("get", [item])
            
    def close(self):
        self.running = False
        self.ws.close()

# Usage example
if __name__ == "__main__":
    agent = DarkPawnsAgent(api_key="YOUR_API_KEY_HERE", player_name="test-agent")
    try:
        agent.connect()
        agent.explore()
        time.sleep(5)
    finally:
        agent.close()

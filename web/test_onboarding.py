#!/usr/bin/env python3
"""
Test script for Dark Pawns agent onboarding content negotiation.
"""

import requests
import json

def test_content_negotiation(base_url="http://localhost:4350"):
    """Test that content negotiation works correctly."""
    
    print("Testing Dark Pawns onboarding content negotiation...")
    print(f"Base URL: {base_url}")
    print()
    
    # Test 1: HTML (default)
    print("1. Testing HTML response (default):")
    response = requests.get(f"{base_url}/onboarding")
    print(f"   Status: {response.status_code}")
    print(f"   Content-Type: {response.headers.get('Content-Type')}")
    print(f"   Length: {len(response.text)} chars")
    print(f"   Is HTML: {'<html>' in response.text[:100].lower()}")
    print()
    
    # Test 2: Markdown
    print("2. Testing Markdown response:")
    response = requests.get(
        f"{base_url}/onboarding",
        headers={"Accept": "text/markdown"}
    )
    print(f"   Status: {response.status_code}")
    print(f"   Content-Type: {response.headers.get('Content-Type')}")
    print(f"   Length: {len(response.text)} chars")
    print(f"   Starts with '#': {response.text.startswith('#')}")
    print(f"   Contains '## Quick Start': {'## Quick Start' in response.text}")
    print()
    
    # Test 3: JSON
    print("3. Testing JSON response:")
    response = requests.get(
        f"{base_url}/onboarding",
        headers={"Accept": "application/json"}
    )
    print(f"   Status: {response.status_code}")
    print(f"   Content-Type: {response.headers.get('Content-Type')}")
    print(f"   Length: {len(response.text)} chars")
    
    try:
        data = json.loads(response.text)
        print(f"   Valid JSON: Yes")
        print(f"   Has @context: {'@context' in data}")
        print(f"   Has messageTypes: {'messageTypes' in data}")
    except json.JSONDecodeError:
        print(f"   Valid JSON: No")
    print()
    
    # Test 4: OpenAPI spec
    print("4. Testing OpenAPI specification:")
    response = requests.get(f"{base_url}/api/openapi.json")
    print(f"   Status: {response.status_code}")
    print(f"   Content-Type: {response.headers.get('Content-Type')}")
    
    try:
        data = json.loads(response.text)
        print(f"   Valid JSON: Yes")
        print(f"   OpenAPI version: {data.get('openapi')}")
        print(f"   Title: {data.get('info', {}).get('title')}")
    except json.JSONDecodeError:
        print(f"   Valid JSON: No")
    print()
    
    # Test 5: Health check
    print("5. Testing health endpoint:")
    response = requests.get(f"{base_url}/health")
    print(f"   Status: {response.status_code}")
    print(f"   Response: {response.text.strip()}")
    print()
    
    print("Content negotiation test complete!")

def generate_agent_code():
    """Generate example agent code from documentation."""
    
    print("Generating example agent code...")
    print()
    
    # Python agent template
    python_code = '''import websocket
import json
import time

class DarkPawnsAgent:
    def __init__(self, api_key, player_name="agent"):
        self.ws = websocket.WebSocket()
        self.api_key = api_key
        self.player_name = player_name
        self.state = {}
        
    def connect(self, url="ws://localhost:4350/ws"):
        self.ws.connect(url)
        return self.login()
        
    def login(self):
        msg = {
            "type": "login",
            "data": {
                "player_name": self.player_name,
                "api_key": self.api_key,
                "mode": "agent"
            }
        }
        self.ws.send(json.dumps(msg))
        response = json.loads(self.ws.recv())
        if response.get("type") == "state":
            self.state = response.get("data", {})
        return response
        
    def command(self, cmd, args=None):
        msg = {
            "type": "command",
            "data": {"command": cmd}
        }
        if args:
            msg["data"]["args"] = args
        self.ws.send(json.dumps(msg))
        response = json.loads(self.ws.recv())
        
        # Update state if received
        if response.get("type") == "state":
            self.state = response.get("data", {})
            
        return response
        
    def explore(self):
        """Basic exploration behavior"""
        print("Starting exploration...")
        
        # Look around
        response = self.command("look")
        room = self.state.get("room", {})
        
        print(f"Room: {room.get('name')}")
        print(f"Exits: {', '.join(room.get('exits', []))}")
        
        # Check for mobs
        mobs = room.get("mobs", [])
        if mobs:
            print(f"Found mobs: {[m['name'] for m in mobs]}")
            # Attack first mob
            target = mobs[0]['name']
            print(f"Attacking {target}...")
            self.command("hit", [target])
            
        # Check for items
        items = room.get("items", [])
        if items:
            print(f"Found items: {[i['name'] for i in items]}")
            # Pick up first item
            item = items[0]['name']
            print(f"Getting {item}...")
            self.command("get", [item])
            
    def close(self):
        self.ws.close()

# Usage example
if __name__ == "__main__":
    agent = DarkPawnsAgent(api_key="YOUR_API_KEY_HERE", player_name="test-agent")
    try:
        agent.connect()
        agent.explore()
    finally:
        agent.close()
'''
    
    print("Python Agent Template:")
    print("=" * 50)
    print(python_code)
    print("=" * 50)
    
    # Save to file
    with open("example_agent.py", "w") as f:
        f.write(python_code)
    print("\nSaved example agent code to 'example_agent.py'")

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1 and sys.argv[1] == "test":
        base_url = sys.argv[2] if len(sys.argv) > 2 else "http://localhost:4350"
        test_content_negotiation(base_url)
    elif len(sys.argv) > 1 and sys.argv[1] == "code":
        generate_agent_code()
    else:
        print("Usage:")
        print("  python test_onboarding.py test [base_url]  # Test content negotiation")
        print("  python test_onboarding.py code             # Generate example agent code")
        print()
        print("Examples:")
        print("  python test_onboarding.py test")
        print("  python test_onboarding.py test http://darkpawns.labz0rz.com")
        print("  python test_onboarding.py code")
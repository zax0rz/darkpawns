#!/usr/bin/env python3
"""
Test script for Dark Pawns agent onboarding content negotiation.
"""

import requests
import json


def test_content_negotiation(base_url="http://localhost:4350"):
    """Test that content negotiation works correctly."""

    # Test 1: HTML (default)
    response = requests.get(f"{base_url}/onboarding")
    assert response.status_code == 200, \
        f"Expected 200, got {response.status_code} for /onboarding"
    content_type = response.headers.get("Content-Type", "")
    assert "text/html" in content_type, \
        f"Expected text/html, got {content_type}"
    assert "<html" in response.text[:100].lower(), \
        "Response should contain HTML markup"

    # Test 2: Markdown
    response = requests.get(
        f"{base_url}/onboarding",
        headers={"Accept": "text/markdown"}
    )
    assert response.status_code == 200, \
        f"Expected 200, got {response.status_code} for markdown"
    ct = response.headers.get("Content-Type", "")
    assert "text/markdown" in ct or "text/plain" in ct, \
        f"Expected text/markdown or text/plain, got {ct}"
    assert response.text.startswith("#"), \
        "Markdown response should start with a heading"
    assert "## Quick Start" in response.text, \
        "Markdown response should contain Quick Start section"

    # Test 3: JSON
    response = requests.get(
        f"{base_url}/onboarding",
        headers={"Accept": "application/json"}
    )
    assert response.status_code == 200, \
        f"Expected 200, got {response.status_code} for JSON"
    ct = response.headers.get("Content-Type", "")
    assert "application/json" in ct, \
        f"Expected application/json, got {ct}"
    data = json.loads(response.text)
    assert "@context" in data, \
        "JSON onboarding should include @context"
    assert "messageTypes" in data, \
        "JSON onboarding should include messageTypes"

    # Test 4: OpenAPI spec
    response = requests.get(f"{base_url}/api/openapi.json")
    assert response.status_code == 200, \
        f"Expected 200, got {response.status_code} for OpenAPI spec"
    ct = response.headers.get("Content-Type", "")
    assert "application/json" in ct or "application/octet-stream" in ct, \
        f"Expected JSON, got {ct}"
    data = json.loads(response.text)
    assert "openapi" in data, \
        "OpenAPI spec should include openapi version field"
    assert "info" in data and "title" in data["info"], \
        "OpenAPI spec should include info.title"

    # Test 5: Health check
    response = requests.get(f"{base_url}/health")
    assert response.status_code == 200, \
        f"Expected 200, got {response.status_code} for health"
    assert response.text.strip() == "OK", \
        f"Expected 'OK', got '{response.text.strip()}'"


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
        print("All content negotiation tests passed!")
    elif len(sys.argv) > 1 and sys.argv[1] == "code":
        generate_agent_code()
    else:
        print("Usage:")
        print("  python test_onboarding.py test [base_url]")
        print("  python test_onboarding.py code")

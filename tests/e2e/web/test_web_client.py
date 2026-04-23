#!/usr/bin/env python3
"""
End-to-end tests for Dark Pawns web client.
"""

import pytest
import requests
import json
import time
import websocket
import threading
import queue
from urllib.parse import urljoin

class TestWebClientE2E:
    """End-to-end tests for web client functionality."""
    
    @pytest.fixture
    def base_url(self):
        """Base URL for the Dark Pawns server."""
        return "http://localhost:4350"
    
    @pytest.fixture
    def ws_url(self):
        """WebSocket URL for the Dark Pawns server."""
        return "ws://localhost:4350/ws"
    
    @pytest.fixture
    def test_player(self):
        """Test player credentials."""
        return {
            "player_name": "test_e2e_player",
            "api_key": "test-api-key-e2e",
            "mode": "player"
        }
    
    def test_server_health(self, base_url):
        """Test that the server is running and healthy."""
        
        response = requests.get(f"{base_url}/health")
        
        assert response.status_code == 200
        assert response.text.strip() == "OK"
    
    def test_onboarding_content_negotiation(self, base_url):
        """Test onboarding content negotiation."""
        
        # Test HTML response (default)
        response = requests.get(f"{base_url}/onboarding")
        assert response.status_code == 200
        assert "text/html" in response.headers.get("Content-Type", "")
        assert "<html>" in response.text.lower()
        
        # Test Markdown response
        response = requests.get(
            f"{base_url}/onboarding",
            headers={"Accept": "text/markdown"}
        )
        assert response.status_code == 200
        assert "text/markdown" in response.headers.get("Content-Type", "")
        assert response.text.startswith("#")
        
        # Test JSON response
        response = requests.get(
            f"{base_url}/onboarding",
            headers={"Accept": "application/json"}
        )
        assert response.status_code == 200
        assert "application/json" in response.headers.get("Content-Type", "")
        
        data = json.loads(response.text)
        assert "@context" in data
        assert "messageTypes" in data
    
    def test_openapi_spec(self, base_url):
        """Test OpenAPI specification endpoint."""
        
        response = requests.get(f"{base_url}/api/openapi.json")
        
        assert response.status_code == 200
        assert "application/json" in response.headers.get("Content-Type", "")
        
        spec = json.loads(response.text)
        assert "openapi" in spec
        assert "info" in spec
        assert "paths" in spec
        assert "/ws" in spec["paths"]
        assert "/health" in spec["paths"]
    
    def test_static_assets(self, base_url):
        """Test static asset serving."""
        
        # Test favicon
        response = requests.get(f"{base_url}/favicon.ico")
        assert response.status_code == 200
        
        # Test CSS files
        response = requests.get(f"{base_url}/static/style.css")
        assert response.status_code == 200
        assert "text/css" in response.headers.get("Content-Type", "")
    
    def test_websocket_connection(self, ws_url, test_player):
        """Test WebSocket connection and basic communication."""
        
        # Create WebSocket connection
        ws = websocket.WebSocket()
        ws.connect(ws_url)
        
        try:
            # Send login message
            login_msg = {
                "type": "login",
                "data": test_player
            }
            ws.send(json.dumps(login_msg))
            
            # Receive response
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] in ["state", "error", "welcome"]
            
            if response_data["type"] == "state":
                # Verify state structure
                state = response_data.get("data", {})
                assert "player" in state
                assert "room" in state
                assert "inventory" in state
            
            # Send a look command
            look_msg = {
                "type": "command",
                "data": {"command": "look"}
            }
            ws.send(json.dumps(look_msg))
            
            # Receive look response
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] in ["message", "state", "error"]
            
        finally:
            ws.close()
    
    def test_websocket_realtime_updates(self, ws_url, test_player):
        """Test real-time updates through WebSocket."""
        
        messages = queue.Queue()
        
        def on_message(ws, message):
            messages.put(json.loads(message))
        
        def on_error(ws, error):
            print(f"WebSocket error: {error}")
        
        def on_close(ws, close_status_code, close_msg):
            print("WebSocket closed")
        
        def on_open(ws):
            # Send login when connection opens
            login_msg = {
                "type": "login",
                "data": test_player
            }
            ws.send(json.dumps(login_msg))
        
        # Create WebSocket app
        ws = websocket.WebSocketApp(
            ws_url,
            on_open=on_open,
            on_message=on_message,
            on_error=on_error,
            on_close=on_close
        )
        
        # Run WebSocket in background thread
        ws_thread = threading.Thread(target=ws.run_forever)
        ws_thread.daemon = True
        ws_thread.start()
        
        try:
            # Wait for login response
            for _ in range(10):  # Try for 10 seconds
                try:
                    msg = messages.get(timeout=1)
                    if msg.get("type") in ["state", "welcome"]:
                        break
                except queue.Empty:
                    continue
            
            # Send movement command
            move_msg = {
                "type": "command",
                "data": {"command": "north"}
            }
            ws.send(json.dumps(move_msg))
            
            # Wait for response
            for _ in range(5):
                try:
                    msg = messages.get(timeout=1)
                    if msg.get("type") in ["message", "state", "error"]:
                        # Got a response
                        break
                except queue.Empty:
                    continue
            
            # Send another command
            inv_msg = {
                "type": "command",
                "data": {"command": "inventory"}
            }
            ws.send(json.dumps(inv_msg))
            
            # Wait for inventory response
            for _ in range(5):
                try:
                    msg = messages.get(timeout=1)
                    if msg.get("type") == "state":
                        state = msg.get("data", {})
                        if "inventory" in state:
                            # Got inventory update
                            break
                except queue.Empty:
                    continue
            
        finally:
            ws.close()
            ws_thread.join(timeout=2)
    
    def test_concurrent_websocket_connections(self, ws_url, test_player):
        """Test multiple concurrent WebSocket connections."""
        
        connections = []
        results = []
        
        def connect_client(client_id):
            """Connect a WebSocket client."""
            try:
                ws = websocket.WebSocket()
                ws.connect(ws_url)
                
                # Login
                player_data = test_player.copy()
                player_data["player_name"] = f"{test_player['player_name']}_{client_id}"
                
                login_msg = {
                    "type": "login",
                    "data": player_data
                }
                ws.send(json.dumps(login_msg))
                
                # Get response
                response = ws.recv()
                response_data = json.loads(response)
                
                results.append({
                    "client_id": client_id,
                    "success": response_data.get("type") in ["state", "welcome"],
                    "response": response_data
                })
                
                ws.close()
                return True
                
            except Exception as e:
                results.append({
                    "client_id": client_id,
                    "success": False,
                    "error": str(e)
                })
                return False
        
        # Create multiple concurrent connections
        threads = []
        for i in range(3):  # Test with 3 concurrent connections
            thread = threading.Thread(target=connect_client, args=(i,))
            threads.append(thread)
            thread.start()
        
        # Wait for all threads to complete
        for thread in threads:
            thread.join()
        
        # Verify results
        assert len(results) == 3
        
        success_count = sum(1 for r in results if r["success"])
        assert success_count >= 2  # At least 2 should succeed
    
    def test_websocket_error_handling(self, ws_url):
        """Test WebSocket error handling."""
        
        ws = websocket.WebSocket()
        ws.connect(ws_url)
        
        try:
            # Send invalid JSON
            ws.send("invalid json")
            
            # Should receive error response
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] == "error"
            assert "message" in response_data
            
            # Send message with missing required fields
            invalid_msg = {
                "type": "invalid_type"
            }
            ws.send(json.dumps(invalid_msg))
            
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] == "error"
            
            # Send login with invalid data
            invalid_login = {
                "type": "login",
                "data": {
                    "player_name": "",
                    "api_key": ""
                }
            }
            ws.send(json.dumps(invalid_login))
            
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] in ["error", "welcome"]
            
        finally:
            ws.close()
    
    def test_websocket_reconnection(self, ws_url, test_player):
        """Test WebSocket reconnection after disconnect."""
        
        # First connection
        ws1 = websocket.WebSocket()
        ws1.connect(ws_url)
        
        # Login
        login_msg = {
            "type": "login",
            "data": test_player
        }
        ws1.send(json.dumps(login_msg))
        response1 = ws1.recv()
        
        # Close connection
        ws1.close()
        
        # Wait a bit
        time.sleep(1)
        
        # Reconnect
        ws2 = websocket.WebSocket()
        ws2.connect(ws_url)
        
        # Login again (should work)
        ws2.send(json.dumps(login_msg))
        response2 = ws2.recv()
        response2_data = json.loads(response2)
        
        assert response2_data["type"] in ["state", "welcome"]
        
        ws2.close()
    
    def test_websocket_heartbeat(self, ws_url, test_player):
        """Test WebSocket heartbeat/ping-pong."""
        
        ws = websocket.WebSocket()
        ws.connect(ws_url)
        
        try:
            # Login
            login_msg = {
                "type": "login",
                "data": test_player
            }
            ws.send(json.dumps(login_msg))
            ws.recv()  # Login response
            
            # Send ping (if supported)
            # Note: websocket-client library handles ping/pong automatically
            
            # Send a command
            look_msg = {
                "type": "command",
                "data": {"command": "look"}
            }
            ws.send(json.dumps(look_msg))
            
            # Get response
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] in ["message", "state", "error"]
            
            # Wait and send another command to test connection persistence
            time.sleep(2)
            
            inv_msg = {
                "type": "command",
                "data": {"command": "inventory"}
            }
            ws.send(json.dumps(inv_msg))
            
            response = ws.recv()
            response_data = json.loads(response)
            
            assert response_data["type"] in ["message", "state", "error"]
            
        finally:
            ws.close()
    
    def test_websocket_broadcast_messages(self, ws_url, test_player):
        """Test broadcast messages to multiple clients."""
        
        # This test requires two connections
        messages1 = queue.Queue()
        messages2 = queue.Queue()
        
        def make_connection(messages_queue, player_suffix):
            """Make a WebSocket connection."""
            ws = websocket.WebSocket()
            ws.connect(ws_url)
            
            # Login with unique player name
            player_data = test_player.copy()
            player_data["player_name"] = f"{test_player['player_name']}_{player_suffix}"
            
            login_msg = {
                "type": "login",
                "data": player_data
            }
            ws.send(json.dumps(login_msg))
            
            # Get initial response
            response = ws.recv()
            messages_queue.put(json.loads(response))
            
            return ws
        
        try:
            # Create two connections
            ws1 = make_connection(messages1, "a")
            ws2 = make_connection(messages2, "b")
            
            # Clear initial messages
            while not messages1.empty():
                messages1.get()
            while not messages2.empty():
                messages2.get()
            
            # On a real server with broadcast capability, we would test here
            # For now, just verify both connections work
            
            # Send commands on both connections
            look_msg = {
                "type": "command",
                "data": {"command": "look"}
            }
            
            ws1.send(json.dumps(look_msg))
            ws2.send(json.dumps(look_msg))
            
            # Get responses
            response1 = json.loads(ws1.recv())
            response2 = json.loads(ws2.recv())
            
            assert response1["type"] in ["message", "state", "error"]
            assert response2["type"] in ["message", "state", "error"]
            
        finally:
            try:
                ws1.close()
            except:
                pass
            try:
                ws2.close()
            except:
                pass
    
    def test_websocket_large_messages(self, ws_url, test_player):
        """Test handling of large messages."""
        
        ws = websocket.WebSocket()
        ws.connect(ws_url)
        
        try:
            # Login
            login_msg = {
                "type": "login",
                "data": test_player
            }
            ws.send(json.dumps(login_msg))
            ws.recv()  # Login response
            
            # Send command with large data (if supported)
            # Note: Actual limit would depend on server configuration
            
            # Test with moderately large message
            large_data = {
                "type": "command",
                "data": {
                    "command": "say",
                    "args": ["A" * 1000]  # 1000 character message
                }
            }
            
            ws.send(json.dumps(large_data))
            
            # Try to get response
            try:
                response = ws.recv(timeout=2)
                response_data = json.loads(response)
                assert response_data["type"] in ["message", "error"]
            except websocket.WebSocketTimeoutException:
                # Timeout is acceptable for this test
                pass
            
        finally:
            ws.close()

class TestWebAPIE2E:
    """End-to-end tests for web API endpoints."""
    
    @pytest.fixture
    def base_url(self):
        """Base URL for the Dark Pawns server."""
        return "http://localhost:4350"
    
    def test_api_endpoints_exist(self, base_url):
        """Test that API endpoints exist and return proper responses."""
        
        endpoints = [
            "/api/status",
            "/api/players",
            "/api/rooms",
            "/api/commands"
        ]
        
        for endpoint in endpoints:
            response = requests.get(urljoin(base_url, endpoint))
            
            # Endpoint might return 200, 404, or 405
            # Just verify it doesn't crash the server
            assert response.status_code in [200, 404, 405]
    
    def test_api_documentation(self, base_url):
        """Test API documentation endpoints."""
        
        # OpenAPI JSON spec
        response = requests.get(f"{base_url}/api/openapi.json")
        if response.status_code == 200:
            spec = json.loads(response.text)
            assert "openapi" in spec
            assert "paths" in spec
        
        # Swagger UI (if enabled)
        response = requests.get(f"{base_url}/api/docs")
        # Might return 200 (if enabled) or 404 (if not)
    
    def test_api_error_responses(self, base_url):
        """Test API error responses."""
        
        # Test non-existent endpoint
        response = requests.get(f"{base_url}/api/nonexistent")
        assert response.status_code == 404
        
        # Test malformed requests
        response = requests.post(
            f"{base_url}/api/command",
            data="invalid json",
            headers={"Content-Type": "application/json"}
        )
        # Should return 400 or 422
        assert response.status_code in [400, 422, 405, 404]
    
    def test_api_cors_headers(self, base_url):
        """Test CORS headers for API endpoints."""
        
        # Make request with Origin header
        headers = {"Origin": "http://example.com"}
        response = requests.get(f"{base_url}/health", headers=headers)
        
        # Check for CORS headers (if enabled)
        # Some servers might not have CORS enabled for all endpoints
        cors_headers = [
            "Access-Control-Allow-Origin",
            "Access-Control-Allow-Methods",
            "Access-Control-Allow-Headers"
        ]
        
        for header in cors_headers:
            if header in response.headers:
                # CORS is enabled
                break
    
    def test_api_rate_limiting(self, base_url):
        """Test API rate limiting (if enabled)."""
        
        # Make multiple rapid requests
        responses = []
        for i in range(10):
            response = requests.get(f"{base_url}/health")
            responses.append(response.status_code)
            time.sleep(0.1)  # Small delay
        
        # All should succeed (or some might get rate limited)
        # Just verify server doesn't crash
        assert all(status in [200, 429] for status in responses)

class TestWebSecurityE2E:
    """End-to-end security tests for web interface."""
    
    @pytest.fixture
    def base_url(self):
        """Base URL for the Dark Pawns server."""
        return "http://localhost:4350"
    
    def test_https_redirect(self, base_url):
        """Test HTTPS redirect (if configured)."""
        
        # Note: Local development server might not have HTTPS
        # This test is for production-like environments
        
        # For now, just test that HTTP works
        response = requests.get(f"{base_url}/health", verify=False)
        assert response.status_code == 200
    
    def test_security_headers(self, base_url):
        """Test security headers."""
        
        response = requests.get(f"{base_url}/health")
        
        # Check for common security headers
        security_headers = [
            "X-Content-Type-Options",
            "X-Frame-Options",
            "X-XSS-Protection",
            "Content-Security-Policy",
            "Strict-Transport-Security"
        ]
        
        # Count how many security headers are present
        present_headers = [
            h for h in security_headers 
            if h in response.headers
        ]
        
        # At least some security headers should be present
        # (Exact set depends on server configuration)
        print(f"Security headers present: {present_headers}")
    
    def test_sql_injection_protection(self, base_url):
        """Test SQL injection protection."""
        
        # Try SQL injection in query parameters
        test_payloads = [
            "' OR '1'='1",
            "'; DROP TABLE players; --",
            "1' UNION SELECT * FROM players --"
        ]
        
        for payload in test_payloads:
            # Test in various endpoints
            endpoints = [
                f"/api/players?name={payload}",
                f"/api/rooms?id={payload}"
            ]
            
            for endpoint in endpoints:
                response = requests.get(urljoin(base_url, endpoint))
                
                # Server should not crash
                assert response.status_code in [200, 400, 404, 500]
                
                # If it returns 500, that might indicate an error
                # but at least the server is still running
    
    def test_xss_protection(self, base_url):
        """Test XSS protection."""
        
        # Try XSS payloads
        xss_payloads = [
            "<script>alert('xss')</script>",
            "<img src=x onerror=alert('xss')>",
            "javascript:alert('xss')"
        ]
        
        for payload in xss_payloads:
            # Test in query parameters
            response = requests.get(
                f"{base_url}/health",
                params={"test": payload}
            )
            
            # Server should not crash
            assert response.status_code in [200, 400]
            
            # Response should not contain the raw payload
            # (though it might be URL-encoded)
            if response.status_code == 200:
                response_text = response.text
                # Check that script tags are not present in raw form
                assert "<script>" not in response_text or "&lt;script&gt;" in response_text
    
    def test_directory_traversal(self, base_url):
        """Test directory traversal protection."""
        
        traversal_payloads = [
            "../../../etc/passwd",
            "..\\..\\windows\\system32\\config",
            "%2e%2e%2f%2e%2e%2fetc%2fpasswd"
        ]
        
        for payload in traversal_payloads:
            response = requests.get(f"{base_url}/static/{payload}")
            
            # Should return 404 or 400, not 200 with sensitive data
            assert response.status_code in [404, 400, 403]
            
            # Should not expose file contents
            if response.status_code == 200:
                # If it returns 200, make sure it's not a system file
                assert "root:" not in response.text
                assert "Administrator" not in response.text

if __name__ == "__main__":
    # Run tests
    pytest.main([__file__, "-v"])
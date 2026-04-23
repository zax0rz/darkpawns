#!/usr/bin/env python3
"""
Test script for Dark Pawns documentation site content negotiation.
Verifies that dual rendering (HTML/Markdown) works correctly.
"""

import requests
import json
import sys
from pathlib import Path

BASE_URL = "http://localhost:1313"  # Hugo dev server
# BASE_URL = "https://darkpawns.labz0rz.com/docs"  # Production

def test_html():
    """Test HTML rendering (default)"""
    print("Testing HTML rendering...")
    response = requests.get(f"{BASE_URL}/")
    
    if response.status_code != 200:
        print(f"  ❌ Failed: HTTP {response.status_code}")
        return False
    
    if "text/html" in response.headers.get("Content-Type", ""):
        print("  ✅ HTML content type correct")
    else:
        print(f"  ❌ Wrong content type: {response.headers.get('Content-Type')}")
        return False
    
    if "<html" in response.text.lower():
        print("  ✅ HTML content detected")
        return True
    else:
        print("  ❌ No HTML content found")
        return False

def test_markdown():
    """Test Markdown rendering via Accept header"""
    print("\nTesting Markdown rendering...")
    headers = {"Accept": "text/markdown"}
    response = requests.get(f"{BASE_URL}/", headers=headers)
    
    if response.status_code != 200:
        print(f"  ❌ Failed: HTTP {response.status_code}")
        return False
    
    if "text/markdown" in response.headers.get("Content-Type", ""):
        print("  ✅ Markdown content type correct")
    else:
        print(f"  ❌ Wrong content type: {response.headers.get('Content-Type')}")
        return False
    
    # Check for markdown content
    if "# " in response.text or "## " in response.text:
        print("  ✅ Markdown content detected")
        return True
    else:
        print("  ❌ No markdown content found")
        return False

def test_openapi():
    """Test OpenAPI specification"""
    print("\nTesting OpenAPI specification...")
    response = requests.get(f"{BASE_URL}/api/openapi.json")
    
    if response.status_code != 200:
        print(f"  ❌ Failed: HTTP {response.status_code}")
        return False
    
    if "application/json" in response.headers.get("Content-Type", ""):
        print("  ✅ JSON content type correct")
    else:
        print(f"  ❌ Wrong content type: {response.headers.get('Content-Type')}")
        return False
    
    try:
        data = response.json()
        if data.get("openapi") == "3.0.0":
            print("  ✅ OpenAPI 3.0.0 specification valid")
            return True
        else:
            print("  ❌ Not a valid OpenAPI spec")
            return False
    except json.JSONDecodeError:
        print("  ❌ Invalid JSON")
        return False

def test_search_index():
    """Test search index"""
    print("\nTesting search index...")
    response = requests.get(f"{BASE_URL}/search-index.json")
    
    if response.status_code != 200:
        print(f"  ❌ Failed: HTTP {response.status_code}")
        return False
    
    if "application/json" in response.headers.get("Content-Type", ""):
        print("  ✅ JSON content type correct")
    else:
        print(f"  ❌ Wrong content type: {response.headers.get('Content-Type')}")
        return False
    
    try:
        data = response.json()
        if isinstance(data, list):
            print(f"  ✅ Search index contains {len(data)} items")
            return True
        else:
            print("  ❌ Search index is not a list")
            return False
    except json.JSONDecodeError:
        print("  ❌ Invalid JSON")
        return False

def test_agent_protocol_page():
    """Test agent protocol page with content negotiation"""
    print("\nTesting agent protocol page...")
    
    # Test HTML
    response = requests.get(f"{BASE_URL}/agents/protocol/")
    if response.status_code != 200:
        print(f"  ❌ HTML failed: HTTP {response.status_code}")
        return False
    
    if "<html" in response.text.lower():
        print("  ✅ HTML version works")
    else:
        print("  ❌ HTML version failed")
        return False
    
    # Test Markdown
    headers = {"Accept": "text/markdown"}
    response = requests.get(f"{BASE_URL}/agents/protocol/", headers=headers)
    if response.status_code != 200:
        print(f"  ❌ Markdown failed: HTTP {response.status_code}")
        return False
    
    if "# WebSocket Protocol Specification" in response.text:
        print("  ✅ Markdown version works")
        return True
    else:
        print("  ❌ Markdown version failed")
        return False

def test_copy_paste_commands():
    """Test that copy/paste commands are present in pages"""
    print("\nTesting copy/paste commands...")
    
    # Get the home page markdown
    headers = {"Accept": "text/markdown"}
    response = requests.get(f"{BASE_URL}/", headers=headers)
    
    if response.status_code != 200:
        print(f"  ❌ Failed to get page: HTTP {response.status_code}")
        return False
    
    content = response.text
    
    # Check for code blocks
    if "```python" in content or "```javascript" in content:
        print("  ✅ Code blocks found")
        
        # Check for specific commands
        if "websocket" in content.lower() and "connect" in content.lower():
            print("  ✅ WebSocket connection examples found")
            return True
        else:
            print("  ❌ No WebSocket examples found")
            return False
    else:
        print("  ❌ No code blocks found")
        return False

def run_all_tests():
    """Run all tests"""
    print("=" * 60)
    print("Dark Pawns Documentation Site - Content Negotiation Tests")
    print("=" * 60)
    
    tests = [
        ("HTML Rendering", test_html),
        ("Markdown Rendering", test_markdown),
        ("OpenAPI Specification", test_openapi),
        ("Search Index", test_search_index),
        ("Agent Protocol Page", test_agent_protocol_page),
        ("Copy/Paste Commands", test_copy_paste_commands),
    ]
    
    results = []
    for name, test_func in tests:
        print(f"\n{name}:")
        try:
            success = test_func()
            results.append((name, success))
        except Exception as e:
            print(f"  ❌ Exception: {e}")
            results.append((name, False))
    
    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    
    passed = 0
    total = len(results)
    
    for name, success in results:
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status}: {name}")
        if success:
            passed += 1
    
    print(f"\nPassed: {passed}/{total} ({passed/total*100:.1f}%)")
    
    if passed == total:
        print("\n🎉 All tests passed! Documentation site is working correctly.")
        return True
    else:
        print(f"\n⚠️  {total - passed} test(s) failed. Check the output above.")
        return False

if __name__ == "__main__":
    # Check if Hugo dev server is running
    try:
        response = requests.get(BASE_URL, timeout=2)
    except requests.exceptions.ConnectionError:
        print(f"⚠️  Hugo dev server not running at {BASE_URL}")
        print("Start it with: cd docs-site && hugo server -D")
        print("\nTesting with built files instead...")
        
        # Test with built files
        public_dir = Path(__file__).parent / "public"
        if not public_dir.exists():
            print("❌ No built files found. Run: cd docs-site && hugo --minify")
            sys.exit(1)
        
        # We can't test content negotiation without a server,
        # but we can verify files exist
        print("\nChecking built files...")
        
        files_to_check = [
            ("index.html", True),
            ("agents/protocol/index.html", True),
            ("api/openapi.json", True),
            ("search-index.json", True),
        ]
        
        all_exist = True
        for file_path, required in files_to_check:
            full_path = public_dir / file_path
            if full_path.exists():
                print(f"  ✅ {file_path}")
            elif required:
                print(f"  ❌ {file_path} (missing)")
                all_exist = False
            else:
                print(f"  ⚠️  {file_path} (optional, missing)")
        
        if all_exist:
            print("\n✅ All required files built successfully.")
            print("\nTo test content negotiation, start the Hugo server:")
            print("  cd docs-site && hugo server -D")
            print("\nThen run this test script again.")
            sys.exit(0)
        else:
            print("\n❌ Some required files are missing.")
            sys.exit(1)
    
    # Run tests if server is running
    success = run_all_tests()
    sys.exit(0 if success else 1)
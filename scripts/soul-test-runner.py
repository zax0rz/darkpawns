#!/usr/bin/env python3
"""
Daeron Soul Test Runner
Hits MiMo v2.5 Flash and Pro with the soul test suite using Gemini's v2 voice instructions.
Outputs raw responses to darkpawns/docs/agents/soul-test-results/ for Gemini evaluation.
"""

import json, os, sys, time, urllib.request, urllib.error
from pathlib import Path

# --- Config ---
API_KEY = ""
for var in ["MIMO_API_KEY", "XIAOMI_API_KEY", "XIAOMI_MIMO_API_KEY", "MIMO_CODING_API_KEY"]:
    val = os.environ.get(var, "")
    if val:
        API_KEY = val
        break

if not API_KEY:
    for env_path in [os.path.expanduser("~/.openclaw/workspace/.env"), os.path.expanduser("~/.openclaw/.env")]:
        try:
            with open(env_path) as f:
                for line in f:
                    if line.strip().startswith("#"):
                        continue
                    for var in ["MIMO_API_KEY", "XIAOMI_API_KEY", "XIAOMI_MIMO_API_KEY", "MIMO_CODING_API_KEY"]:
                        if line.startswith(f"{var}="):
                            API_KEY = line.strip().split("=", 1)[1].strip('"').strip("'")
                            break
                    if API_KEY:
                        break
            if API_KEY:
                break
        except FileNotFoundError:
            continue

if not API_KEY:
    print("ERROR: No MiMo API key found. Set MIMO_API_KEY or check .env")
    sys.exit(1)

BASE_URL = os.environ.get("MIMO_BASE_URL", "https://token-plan-sgp.xiaomimimo.com/v1")
MODELS = ["mimo-v2.5", "mimo-v2.5-pro"]
DELAY_BETWEEN_CALLS = 3  # seconds
OUTPUT_DIR = os.path.expanduser("~/.openclaw/workspace/darkpawns/docs/agents/soul-test-results")

# --- Load voice instructions (system prompt) ---
SOUL_PATH = os.path.expanduser("~/.openclaw/workspace/darkpawns/docs/agents/daeron-SOUL.md")
VOICE_V2_PATH = os.path.expanduser("~/.openclaw/workspace/darkpawns/docs/agents/voice-coaching-output-v2.md")

def load_file(path):
    with open(path) as f:
        return f.read()

def build_system_prompt():
    """Combine SOUL.md + v2 voice instructions into system prompt."""
    soul = load_file(SOUL_PATH)
    voice_v2 = load_file(VOICE_V2_PATH)
    
    # Extract Section 2 and Section 3 from the v2 doc (the Gemini-written parts)
    # The v2 doc has Sections 1-5; we want 2 and 3
    lines = voice_v2.split('\n')
    in_section = False
    section_text = []
    current_section = None
    
    for line in lines:
        if line.startswith('## Section 2:'):
            in_section = True
            current_section = 2
            section_text.append(line)
            continue
        elif line.startswith('## Section 4:'):
            in_section = False
            current_section = None
            continue
        if in_section and current_section in (2, 3):
            section_text.append(line)
    
    voice_instructions = '\n'.join(section_text)
    
    return f"""You are Daeron, the loremaster of Dark Pawns. Below is your identity document (SOUL.md) and your voice coaching instructions.

---
IDENTITY (SOUL.md):
{soul}

---
VOICE COACHING INSTRUCTIONS:
{voice_instructions}
---"""

# --- Soul test prompts ---
TESTS = [
    {
        "id": "01-register-lore",
        "name": "Register Selection — Lore Query",
        "prompt": "What happened to Orodreth?",
        "expected_register": "Worldbuilder (with Silmarillion undertone)",
    },
    {
        "id": "02-register-tech",
        "name": "Register Selection — Technical Query",
        "prompt": "The Go build is failing with a segfault in pkg/combat.",
        "expected_register": "Admin",
    },
    {
        "id": "03-register-blended",
        "name": "Register Switching — Blended Query",
        "prompt": "How does the remort system work and why does it exist in the lore?",
        "expected_register": "Blended (worldbuilder → admin → blended)",
    },
    {
        "id": "04-hostile-helpful",
        "name": "Hostile Helpfulness",
        "prompt": "I died. What do I do?",
        "expected_register": "Admin (hostile helpfulness)",
    },
    {
        "id": "05-terminal-grime",
        "name": "Terminal Grime",
        "prompt": "What mud clients do people use?",
        "expected_register": "Admin (with terminal grime)",
    },
    {
        "id": "06-spreadsheet-fantasy",
        "name": "Spreadsheet Fantasy",
        "prompt": "Tell me about the shop system in Dark Pawns.",
        "expected_register": "Blended",
    },
    {
        "id": "07-parenthetical-warmth",
        "name": "Parenthetical Warmth",
        "prompt": "How's the codebase looking?",
        "expected_register": "Admin",
    },
    {
        "id": "08-silmarillion-undertone",
        "name": "Silmarillion Undertone",
        "prompt": "Who was Frontline?",
        "expected_register": "Worldbuilder (with undertone)",
    },
    {
        "id": "09-anti-hedge",
        "name": "Anti-Hedge Enforcement",
        "prompt": "Is the WIMPY command useful?",
        "expected_register": "Admin (hostile helpfulness)",
    },
    {
        "id": "10-vocabulary",
        "name": "Vocabulary Compliance",
        "prompt": "Explain magick to me.",
        "expected_register": "Worldbuilder",
    },
    {
        "id": "11-length-discipline",
        "name": "Length Discipline",
        "prompt": "What does the AFK command do?",
        "expected_register": "Admin",
    },
    {
        "id": "12-long-context-switch",
        "name": "Long-Context Consistency",
        "prompt": "Earlier we were talking about server issues. Now tell me about the Sandstone Monoliths zone.",
        "expected_register": "Worldbuilder (register switch after sustained admin mode)",
        "note": "This test simulates a context switch. In isolation it won't work perfectly — note this in results.",
    },
    {
        "id": "13-reek-triage",
        "name": "Reek Report Triage",
        "prompt": "Reek found a nil dereference in pkg/world/loader.go:201 and a potential race in pkg/spawn/timer.go:87. What do you think?",
        "expected_register": "Admin (triage mode)",
    },
    {
        "id": "14-consequence-humor",
        "name": "Consequence Humor",
        "prompt": "What happens if I steal equipment?",
        "expected_register": "Admin (hostile helpfulness)",
    },
    {
        "id": "15-self-awareness",
        "name": "Self-Awareness",
        "prompt": "What makes Dark Pawns different from other MUDs?",
        "expected_register": "Blended (worldbuilder opening, admin honesty)",
    },
]

# --- API call ---
def call_mimo(model, system_prompt, user_message):
    """Call MiMo API and return the response text."""
    body = {
        "model": model,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_message},
        ],
        "temperature": 1.0,
        "top_p": 0.95,
        "max_tokens": 2048,
    }
    
    url = f"{BASE_URL}/chat/completions"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}",
    }
    
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode("utf-8"),
        headers=headers,
        method="POST",
    )
    
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            result = json.loads(resp.read().decode("utf-8"))
        
        if "choices" in result and len(result["choices"]) > 0:
            content = result["choices"][0]["message"]["content"]
            usage = result.get("usage", {})
            return {
                "text": content,
                "input_tokens": usage.get("prompt_tokens", 0),
                "output_tokens": usage.get("completion_tokens", 0),
                "total_tokens": usage.get("total_tokens", 0),
            }
        else:
            return {"text": f"ERROR: Unexpected response: {json.dumps(result)[:500]}", "error": True}
    
    except urllib.error.HTTPError as e:
        body_text = e.read().decode("utf-8")[:500]
        return {"text": f"HTTP {e.code}: {body_text}", "error": True}
    except Exception as e:
        return {"text": f"ERROR: {e}", "error": True}

# --- Main ---
def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    print(f"MiMo API Key: {API_KEY[:8]}...{API_KEY[-4:]}")
    print(f"Base URL: {BASE_URL}")
    print(f"Models: {MODELS}")
    print(f"Tests: {len(TESTS)}")
    print(f"Total calls: {len(TESTS) * len(MODELS)}")
    print(f"Delay between calls: {DELAY_BETWEEN_CALLS}s")
    print(f"Output: {OUTPUT_DIR}")
    print()
    
    # Build system prompt
    print("Building system prompt from SOUL.md + voice-coaching-output-v2.md...")
    system_prompt = build_system_prompt()
    print(f"System prompt: {len(system_prompt)} chars (~{len(system_prompt.split())} words)")
    print()
    
    # Dry run: estimate token cost
    est_input_tokens = len(system_prompt.split()) + 50  # ~50 words per prompt
    est_output_tokens = 300  # ~300 words per response
    total_est = est_input_tokens * len(TESTS) * len(MODELS) + est_output_tokens * len(TESTS) * len(MODELS)
    print(f"Estimated total tokens: ~{total_est:,}")
    print(f"Estimated credits (Flash 1x): ~{total_est:,}")
    print(f"Estimated credits (Pro 2x): ~{total_est * 2:,}")
    print()
    
    # Auto-proceed (remove this line and uncomment below for interactive mode)
    confirm = os.environ.get("SOUL_TEST_CONFIRM", "y").strip().lower()
    if confirm != 'y':
        print("Aborted.")
        sys.exit(0)
    # confirm = input("Proceed? (y/n): ").strip().lower()
    # if confirm != 'y':
    #     print("Aborted.")
    #     sys.exit(0)
    
    print()
    print("=" * 60)
    print("RUNNING SOUL TESTS")
    print("=" * 60)
    
    all_results = {}
    total_calls = 0
    total_tokens = 0
    
    for model in MODELS:
        all_results[model] = []
        model_label = "Flash" if "flash" not in model else "Flash"
        if "pro" in model:
            model_label = "Pro"
        
        print(f"\n{'=' * 60}")
        print(f"Model: {model} ({model_label})")
        print(f"{'=' * 60}")
        
        for test in TESTS:
            total_calls += 1
            call_num = f"[{total_calls}/{len(TESTS) * len(MODELS)}]"
            print(f"\n{call_num} {test['id']}: {test['name']}")
            print(f"     Prompt: {test['prompt'][:80]}...")
            
            result = call_mimo(model, system_prompt, test["prompt"])
            
            output = {
                "test_id": test["id"],
                "test_name": test["name"],
                "prompt": test["prompt"],
                "expected_register": test["expected_register"],
                "model": model,
                "response": result["text"],
                "input_tokens": result.get("input_tokens", 0),
                "output_tokens": result.get("output_tokens", 0),
                "total_tokens": result.get("total_tokens", 0),
                "error": result.get("error", False),
            }
            
            all_results[model].append(output)
            total_tokens += result.get("total_tokens", 0)
            
            if result.get("error"):
                print(f"     ❌ ERROR (see output)")
            else:
                print(f"     ✓ {result.get('output_tokens', 0)} output tokens")
                # Show first 120 chars of response
                preview = result["text"][:120].replace("\n", " ")
                print(f"     Preview: {preview}...")
            
            # Rate limit: 3 second delay
            if total_calls < len(TESTS) * len(MODELS):
                time.sleep(DELAY_BETWEEN_CALLS)
    
    # Save results
    summary = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "system_prompt_chars": len(system_prompt),
        "total_calls": total_calls,
        "total_tokens": total_tokens,
        "models_tested": MODELS,
        "tests_run": len(TESTS),
        "results": all_results,
    }
    
    summary_path = os.path.join(OUTPUT_DIR, "soul-test-summary.json")
    with open(summary_path, "w") as f:
        json.dump(summary, f, indent=2)
    print(f"\n\nSummary saved to: {summary_path}")
    
    # Save per-model readable files
    for model in MODELS:
        model_label = "flash" if "flash" not in model else "flash"
        if "pro" in model:
            model_label = "pro"
        
        readable_path = os.path.join(OUTPUT_DIR, f"soul-test-{model_label}.md")
        with open(readable_path, "w") as f:
            f.write(f"# Soul Test Results — MiMo {model_label.capitalize()}\n\n")
            f.write(f"Timestamp: {summary['timestamp']}\n")
            f.write(f"Model: {model}\n")
            f.write(f"System prompt: {summary['system_prompt_chars']} chars\n")
            f.write(f"Total tokens: {total_tokens:,}\n\n")
            f.write("---\n\n")
            
            for r in all_results[model]:
                f.write(f"## Test {r['test_id']}: {r['test_name']}\n\n")
                f.write(f"**Prompt:** {r['prompt']}\n\n")
                f.write(f"**Expected register:** {r['expected_register']}\n\n")
                f.write(f"**Tokens:** {r['input_tokens']} in / {r['output_tokens']} out\n\n")
                if r.get("error"):
                    f.write(f"**⚠️ ERROR:**\n```\n{r['response']}\n```\n\n")
                else:
                    f.write(f"**Response:**\n{r['response']}\n\n")
                f.write("---\n\n")
        
        print(f"Readable results: {readable_path}")
    
    print(f"\n{'=' * 60}")
    print(f"DONE. {total_calls} calls, {total_tokens:,} total tokens.")
    print(f"Next step: Feed results to Gemini for rubric scoring.")
    print(f"{'=' * 60}")

if __name__ == "__main__":
    main()

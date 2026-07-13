# Dark Pawns — Client Rendering Fixes (2 Issues)

**Target file:** `web/client.js` (331 lines, vanilla JS, xterm.js terminal)
**No server changes needed.** Both bugs are client-side rendering.

**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/client-char-create-rendering`
**After fixing:** Open `web/index.html` in a browser, connect to a running instance, start character creation and verify menus render with proper line breaks and option labels.
**Push:** `git push origin fix/client-char-create-rendering`

---

## Bug Context

The Go server sends `MsgCharCreate` messages as JSON with this shape:
```json
{
  "type": "char_create",
  "data": {
    "stage": "race",
    "prompt": "Race Menu Text Here\r\nRace: ",
    "options": [
      {"Key": "H", "Label": "Human"},
      {"Key": "E", "Label": "Elven"},
      {"Key": "D", "Label": "Dwarven"}
    ]
  }
}
```

The prompt text contains `\r\n` for line breaks. The options are a **JSON array** of `{Key, Label}` objects (Go type: `[]CharCreateOption`).

---

## Fix 1: Options render as `[object Object]` (DP-1063) — CRITICAL

**File:** `web/client.js`
**Lines:** 213-222

**Current code:**
```javascript
// Show numbered options if present
if (msg.data && msg.data.options && typeof msg.data.options === 'object') {
    const opts = msg.data.options;
    const keys = Object.keys(opts);
    if (keys.length > 0 && typeof opts[keys[0]] === 'string') {
        keys.forEach(function (k) {
            term.writeln('  \x1b[33m' + k + '\x1b[0m) ' + opts[k]);
        });
    }
}
```

**Problem:** `Object.keys()` on a JSON array returns array indices `["0", "1", "2"]`. `opts["0"]` is the object `{Key: "H", Label: "Human"}`, not a string. The `typeof ... === 'string'` check fails, so options are silently skipped. The `[object Object]` text the user sees comes from the prompt text that was supposed to have been pre-rendered with options embedded — but it wasn't.

**Fix:** Add `Array.isArray()` check before the existing `Object.keys()` path:
```javascript
// Show numbered options if present
if (msg.data && msg.data.options && typeof msg.data.options === 'object') {
    const opts = msg.data.options;
    if (Array.isArray(opts)) {
        // Server sends []CharCreateOption: [{Key, Label}, ...]
        opts.forEach(function (o) {
            term.writeln('  \x1b[33m' + o.Key + '\x1b[0m) ' + o.Label);
        });
    } else {
        // Fallback: plain {key: label} map
        const keys = Object.keys(opts);
        if (keys.length > 0 && typeof opts[keys[0]] === 'string') {
            keys.forEach(function (k) {
                term.writeln('  \x1b[33m' + k + '\x1b[0m) ' + opts[k]);
            });
        }
    }
}
```

**Expected result:** Character creation menus (Y/N, race, class, hometown, stats) render as:
```
  H) Human
  E) Elven
  D) Dwarven
  K) Kenderkin
  M) Minotaur
  R) Rakshasan
  S) Ssauran
Race: 
```

---

## Fix 2: `\r\n` not rendered as line breaks in prompts (DP-1064) — CRITICAL

**File:** `web/client.js`
**Line:** 210 (inside the `char_create` handler) AND line ~248 (the final `term.writeln(text)` at the bottom of the message handler)

**Problem:** xterm.js `writeln()` adds its own `\n`. When prompt text contains `\r\n`, the `\r` is a carriage return (position reset, no visible effect in xterm) but the `\n` pairs with `writeln()`'s added `\n` to create **double-spaced** output. Menu text like:
```
Race Menu\r\n  H) Human\r\n  E) Elven\r\n
```
Renders as a single long wrapped line because the JSON parser may not preserve the literal `\r\n` — it depends on how the JSON string was constructed on the server side.

**Root cause (verify first):** The Go server sends the prompt with literal `\r\n` bytes (Go strings contain actual newline/carriage-return characters, not escape sequences). When JSON-encoded, these become `\r\n` in the JSON string. `JSON.parse()` on the client converts them back to real `\r` and `\n` characters. xterm's `writeln()` handles `\n` fine. So `\r` + `\n` = `\r` (harmless position reset) + `\n` (actual newline) + `writeln` adds another `\n` = double spacing.

**Fix option A (simpler, recommended):** Strip `\r` from prompt text before writing. This prevents double-spacing from the `writeln()` + `\n` combination:
```javascript
if (msg.data && msg.data.prompt) {
    term.writeln(msg.data.prompt.replace(/\r/g, ''));
}
```

**Fix option B (more precise):** Use `term.write()` instead of `term.writeln()` for prompts that already contain their own newlines. This avoids the extra `\n`:
```javascript
if (msg.data && msg.data.prompt) {
    term.write(msg.data.prompt.replace(/\r/g, ''));
}
```

**Also apply to the fallback text handler at the bottom of `ws.onmessage`:**
```javascript
if (text) term.writeln(text.replace(/\r/g, ''));
```

**Use option B** — `term.write()` without the auto-newline is the right call for prompts that already end with `\n` (most of them do since the Go server appends `\r\n`). The fallback text handler at the bottom should use `term.writeln()` but still strip `\r` to prevent double-spacing.

---

## Summary

| # | Issue | Lines | Change |
|---|-------|-------|--------|
| 1 | DP-1063 | 213-222 | Add `Array.isArray()` path to options renderer |
| 2 | DP-1064 | 210, ~248 | Strip `\r`, use `term.write()` for prompts |

**Commit message:** `fix: client char_create rendering — options array support, line breaks (DP-1063 DP-1064)`

**Testing:** No automated tests exist for `web/client.js`. Manual verification required:
1. Open `web/index.html` in browser
2. Create a new character through the full flow
3. Verify: race menu shows labeled options on separate lines
4. Verify: class menu shows labeled options on separate lines
5. Verify: hometown menu shows labeled options on separate lines
6. Verify: Y/N prompts show `[Y] Yes [N] No` style options
7. Verify: no `[object Object]` text anywhere
8. Verify: no excessive blank lines between menu items

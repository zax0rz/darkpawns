# entry.new-password

R1/R5e/R5g: src/interpreter.c:1187-1190,1721 removes leading C whitespace before every nanny state; :1942-1980 owns the length/name, mismatch and echo gates; src/structs.h:648 bounds new secrets to ten bytes. Go now removes only leading ASCII C whitespace on creation/confirmation, retaining trailing bytes. Gate and ordinary mismatch proofs cover 0/1/2/3/10/11-byte secrets, folded names, whitespace, secret flags, and both real TCP and WebSocket JSON boundaries.

Retained raw targeted run: ~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-entry-password-gates-raw, CLEAN (1/1). Its C blocks explicitly print every intended refusal/confirmation; keep-prompts preserves CRLF. Initial creation-section attempt is retained in dp-1371-entry-password-gates: the existing setup refusal guard rejected intentionally invalid inputs despite eventual world entry. The final vehicle leaves setup at name confirmation and probes the gate directly; no harness change.

R5h whitespace mutation: new-password [0,1,0], assertion failure, retained in dp-1371-entry-train-proofs/new-password; replay revert_proofs.py. All build/vet/test/lint/depth/unit/string gates passed, logs in new-password-gates.

Other readers: charPassword is temporary plaintext until confirmation, then bcrypt; persistAcceptedCharacter copies only the hash into the durable row and menuPasswordHash. Resend prompts preserve secret flags. The terminal bridge and browser JSON both dispatch the same handleCharInput; browser rendering is separate. Login/menu password comparisons are audited in later train cases. No admin or agent credential storage changes.

## Pending decision: password comparison policy

C CRYPT in src/utils.h:623-627 uses crypt when available; the reference config has HAVE_CRYPT=1. Character-name salt selects DES, which compares the first eight password bytes. Go bcrypt compares the complete secret. The existing docs/briefs/2026-07-12-con-menu.md explicitly requires bcrypt and forbids DES. No DP-numbered fidelity approval for the observable suffix-equivalence departure was found. Keep bcrypt and the parent row blocked, with proven gate subrows, until Zach approves an exact divergent-approved row. Proposed scope: retain full-secret bcrypt comparisons across creation confirmation, login, menu password changes/deletion, and admin auth; do not truncate authentication secrets to reproduce DES. No security policy changed in this commit.

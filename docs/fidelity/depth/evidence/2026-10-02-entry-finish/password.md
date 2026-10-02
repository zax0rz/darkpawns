# Full-secret comparison (DP-1380)

Zach approved retaining full-secret bcrypt comparison. R1/R5g: C CRYPT is crypt at src/utils.h:623-626; entry callsites src/interpreter.c:1876,1950,1964,2292,2307. The reference DES equivalence stops at eight bytes. The new divergent-approved row separates this decision from C password gate proofs; no production hashing or authentication changed.

TestEntryFullPasswordComparison tests matching and same-eight-byte-prefix/different-suffix secrets through creation confirmation, stored SQLite login, menu old password, menu new confirmation and deletion verification. TestAdminFullPasswordComparison tests both HTTP outcomes. R5h replay changes creation confirmation to compare only eight bytes and fails the suffix assertion; fixed/reverted/restored logs are retained under dp-1371-entry-finish-proofs/password-prefix.

Other readers: all bcrypt compare sites were enumerated in session_login.go, menu.go and admin/login.go, plus creation plaintext confirmation before hashing. Stored hashes and security trackers are unchanged. Admin keeps its indistinguishable failure response and timing decoy. Ordinary C-parity scenarios intentionally avoid DES-equivalent distinct suffixes.

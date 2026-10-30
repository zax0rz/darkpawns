# Docs media: review pass 2 (2026-10-30)

PR #2229 retains the existing capture pipeline and adds focused webOLC captures for objects, mobs, zones and Lua scripts, plus the host configuration screen. The room editor capture remains shared. Steps and crops were checked against the running preview; the configuration caption identifies checkout mode.

The notification product examples are Avery again. The capture-only test key remains fixed to keep the recorded fingerprint and signed-archive digest repeatable. `scripts/fixtures/README.md` explicitly labels it a public test key, trusted by nothing in production. `TestDocsCaptureKeyIsSeparateFromPublisher` verifies the pair and rejects equality with the pinned production publisher key. An isolated Go overlay substituting the fixture public key for the production pin made that test fail with “documentation capture key must never be the production publisher key”; the production file was untouched.

The native Mudlet capture profile disables the Short map-information contributor: that overlay exposes a room number even with map room labels disabled. The map remains visible and the game transcript is unchanged (R1). Two native Mudlet captures matched SHA-256 `63deb30766b559a2234c58d222b8ffbe0305f3ae298f1bb6629d9aa725fc0b6f`. Two configuration captures matched `69766008b284f5030faff4392ab34b1a437051f26e23e2d57017ec55ae5a279a`. The four new webOLC PNGs and room-editor PNG also matched across two focused runs. These are subset comparisons, not a new claim that every complete-run GIF is byte-identical.

The CLI capture now rebuilds the admin UI even when dist already exists, so the portable bundle can't silently retain earlier product copy. The checker requires all 19 outputs. The complete capture was regenerated after these changes.

Validation logs and mutation proof are retained under `/home/zach/Archives/darkpawns/docs-pass-2/`; native comparison pairs are under `/home/zach/Archives/docs-pass-2/`. Build, vet, tests, lint, formatting and deadcode gates passed independently. No game behavior changes; no oracle census or deployment.

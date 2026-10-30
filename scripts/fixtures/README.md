# Documentation capture fixtures

`docs-capture-test-private.pem` is a **test key, trusted by nothing** in a production build. Its matching public key is `docs-capture-test-public.pem`. Both are intentionally public fixtures; never use them to publish a real release.

The media script replaces the compiled publisher key only in its disposable capture CLI through a Go build overlay. Keeping this test pair fixed keeps the publisher fingerprint and candidate archive digest stable in the terminal recordings. Generating a new key on each capture would change both.

`TestDocsCaptureKeyIsSeparateFromPublisher` checks that the pair matches and that its public key differs from the pinned production publisher key. The capture script doesn't write or replace the production key.

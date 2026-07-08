#!/usr/bin/env python3
"""
Regression test for DP-821.

batch_classify() used to build its default per-text contexts with
``[{}] * len(texts)``, which creates N references to a single shared dict
instead of N independent dicts. Any in-place mutation of one context would
silently leak into all the others.

Hermetic: no network calls, no real API key required. The LLM call
(classify) and time.sleep() are mocked out.
"""

import os
import sys
import time
from unittest import mock

# Add repo root and this directory to the path so the module under test
# can be imported both as `scripts.emotion_llm_classifier` and as a bare
# module, matching the import fallback pattern used elsewhere in scripts/.
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

try:
    from scripts.emotion_llm_classifier import LLMEmotionClassifier
except ImportError:
    from emotion_llm_classifier import LLMEmotionClassifier


def test_batch_classify_default_contexts_are_independent():
    """batch_classify(texts) with contexts=None must give each text its own
    dict, not N references to one shared dict (DP-821)."""

    # LLMEmotionClassifier.__init__ requires LITELLM_KEY to be set; supply a
    # dummy value so construction succeeds without touching real credentials.
    os.environ.setdefault("LITELLM_KEY", "test-key-dp821")

    classifier = LLMEmotionClassifier(model="minimax-m2.7", use_fallback=False)

    captured_contexts = []

    def fake_classify(text, context=None):
        captured_contexts.append(context)
        return {"category": "neutral", "intensity": 1, "confidence": 0.5, "method": "fake"}

    # Mock out the real classify() call and time.sleep() so this test makes
    # no network calls and doesn't wait on the per-item rate-limit delay.
    with mock.patch.object(classifier, "classify", side_effect=fake_classify), \
         mock.patch.object(time, "sleep", return_value=None):
        classifier.batch_classify(["a", "b", "c"])

    assert len(captured_contexts) == 3, f"Expected 3 contexts, got {len(captured_contexts)}"
    assert captured_contexts[0] is not captured_contexts[1], (
        "contexts[0] and contexts[1] are the same object - shared dict bug (DP-821)"
    )
    assert captured_contexts[1] is not captured_contexts[2], (
        "contexts[1] and contexts[2] are the same object - shared dict bug (DP-821)"
    )

    captured_contexts[0]["mutated"] = True
    assert "mutated" not in captured_contexts[1], (
        "Mutating contexts[0] leaked into contexts[1] - shared dict bug (DP-821)"
    )
    assert "mutated" not in captured_contexts[2], (
        "Mutating contexts[0] leaked into contexts[2] - shared dict bug (DP-821)"
    )


if __name__ == "__main__":
    test_batch_classify_default_contexts_are_independent()
    print("OK")

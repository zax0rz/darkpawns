"""Contract tests for privacy_filter_api — no model required.

These pin the HTTP wire contract the Go client depends on
(pkg/privacy/client.go) and the category-subset / keep_length behavior of
the span applier, using a fake redactor in place of the ~3 GB checkpoint.
The real model path is exercised separately, by an operator who has the
service running:

    PRIVACY_FILTER_URL=http://localhost:8001 make privacy-test

Run these with: python -m pytest deployment/test_privacy_filter_api.py
(from the deployment directory, or with it on PYTHONPATH).
"""

from types import SimpleNamespace

from fastapi.testclient import TestClient

import privacy_filter_api as api

EMAIL = "alice@example.com"
PHONE = "555-867-5309"


class FakeOPF:
    """Stands in for opf.OPF: detects the canned email and phone anywhere
    they appear in the input, like the real model would."""

    def redact(self, text: str):
        spans = []
        for needle, label in ((EMAIL, "private_email"), (PHONE, "private_phone")):
            idx = text.find(needle)
            if idx >= 0:
                spans.append(SimpleNamespace(label=label, start=idx, end=idx + len(needle)))
        return SimpleNamespace(text=text, detected_spans=spans)


def make_client(factory=None) -> TestClient:
    app = api.create_app(factory or (lambda: api.Redactor(FakeOPF())))
    return TestClient(app)


def test_health_reports_healthy():
    with make_client() as client:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json() == {"status": "healthy"}


def test_categories_lists_wire_names():
    with make_client() as client:
        resp = client.get("/categories")
        assert resp.status_code == 200
        assert resp.json() == {"categories": list(api.WIRE_TO_OPF)}


def test_filter_without_categories_redacts_everything():
    with make_client() as client:
        resp = client.post(
            "/filter",
            json={"text": f"mail {EMAIL} call {PHONE}", "config": {}},
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["error"] is None
        assert body["filtered_text"] == "mail [REDACTED] call [REDACTED]"
        assert body["detected_categories"] == ["email", "phone"]


def test_filter_honors_category_subset():
    with make_client() as client:
        resp = client.post(
            "/filter",
            json={
                "text": f"mail {EMAIL} call {PHONE}",
                "config": {"categories": ["email"]},
            },
        )
        body = resp.json()
        assert body["error"] is None
        assert EMAIL not in body["filtered_text"]  # email redacted
        assert PHONE in body["filtered_text"]  # phone survives the subset
        assert body["detected_categories"] == ["email"]


def test_filter_keep_length_masks_with_asterisks():
    with make_client() as client:
        resp = client.post(
            "/filter",
            json={
                "text": f"mail {EMAIL}",
                "config": {"categories": ["email"], "keep_length": True},
            },
        )
        body = resp.json()
        assert body["filtered_text"] == "mail " + "*" * len(EMAIL)
        assert len(body["filtered_text"]) == len("mail " + EMAIL)


def test_filter_custom_replacement():
    with make_client() as client:
        resp = client.post(
            "/filter",
            json={
                "text": f"mail {EMAIL}",
                "config": {"categories": ["email"], "replacement": "[XX]"},
            },
        )
        assert resp.json()["filtered_text"] == "mail [XX]"


def test_unknown_category_returns_error_not_silence():
    with make_client() as client:
        resp = client.post(
            "/filter",
            json={"text": "hello", "config": {"categories": ["nope"]}},
        )
        body = resp.json()
        assert body["error"] == "unknown category: nope"
        assert body["filtered_text"] == "hello"
        assert body["detected_categories"] == []


def test_inference_failure_preserves_text_and_reports_error():
    class ExplodingOPF:
        def redact(self, text: str):
            raise RuntimeError("inference exploded")

    with make_client(lambda: api.Redactor(ExplodingOPF())) as client:
        resp = client.post("/filter", json={"text": "hello", "config": {}})
        body = resp.json()
        assert body["error"] == "redaction failed: inference exploded"
        assert body["filtered_text"] == "hello"


def test_unbuildable_model_makes_health_unhealthy():
    def broken_factory():
        raise RuntimeError("checkpoint missing")

    with make_client(broken_factory) as client:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json()["status"] == "unhealthy"
        assert "checkpoint missing" in resp.json()["error"]

        # /filter still answers with the client-recognized error shape
        # instead of hanging or 500ing.
        resp = client.post("/filter", json={"text": "hello", "config": {}})
        body = resp.json()
        assert body["error"] is not None
        assert body["filtered_text"] == "hello"


def test_apply_spans_rejects_overlaps_and_out_of_range():
    text = "abcdef"
    # Overlapping and bogus spans must be skipped, not corrupt the output.
    out = api.apply_spans(text, [(1, 3), (2, 4), (99, 100), (4, 3)], "[R]", False)
    assert out == "a[R]def"

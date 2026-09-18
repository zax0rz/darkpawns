"""Native privacy-filter service for Dark Pawns.

Serves the HTTP contract pinned by the Go client in pkg/privacy/client.go:

    POST /filter   {"text": "...", "config": {"categories": [...],
                                             "replacement": "...",
                                             "keep_length": bool}}
             ->    {"filtered_text": "...", "detected_categories": [...],
                    "error": "..."}

Redaction is OpenAI's Privacy Filter model (Apache 2.0) through its `opf`
package. The service runs on the operator's host next to the game server;
text never leaves the machine.

Running it (see docs/operational/privacy-filter.md for the full recipe):

    uvicorn privacy_filter_api:app --host 127.0.0.1 --port 8001

Model weights are NOT downloaded by this service. The `opf` package resolves
its checkpoint from OPF_CHECKPOINT or downloads to ~/.opf/privacy_filter
(~3 GB BF16) the first time the model is constructed — which happens at
startup here, so the download is visible in the service logs before the
server starts sending text.
"""

from __future__ import annotations

import os
import threading
from contextlib import asynccontextmanager
from typing import Any, Callable, List, Optional

from fastapi import FastAPI
from pydantic import BaseModel

# Wire categories are the names the Go client knows (pkg/privacy/client.go);
# opf span labels are what the model emits. The private_* labels collapse to
# the public wire names one-to-one.
WIRE_TO_OPF = {
    "account_number": "account_number",
    "address": "private_address",
    "email": "private_email",
    "person": "private_person",
    "phone": "private_phone",
    "url": "private_url",
    "date": "private_date",
    "secret": "secret",
}
OPF_TO_WIRE = {opf_label: wire for wire, opf_label in WIRE_TO_OPF.items()}

# Env knobs. HOST/PORT default to the loopback pair the docs and
# `make privacy-test` use (127.0.0.1:8001); the old default was the Compose
# hostname privacy-filter:8000, which cannot resolve in a native install.
HOST = os.environ.get("PRIVACY_FILTER_API_HOST", "127.0.0.1")
PORT = int(os.environ.get("PRIVACY_FILTER_API_PORT", "8001"))
DEVICE = os.environ.get("PRIVACY_FILTER_DEVICE", "cpu")


class FilterConfig(BaseModel):
    categories: Optional[List[str]] = None
    replacement: str = "[REDACTED]"
    keep_length: bool = False


class FilterRequest(BaseModel):
    text: str
    config: FilterConfig


class FilterResponse(BaseModel):
    filtered_text: str
    detected_categories: List[str]
    error: Optional[str] = None


class Redactor:
    """Holds one OPF instance and serializes access to it.

    CPU inference is single-model; the lock keeps concurrent requests queued
    rather than interleaving inside the runtime. `error` is set when the
    model could not be constructed, so /health and /filter can say why.
    """

    def __init__(self, opf: Any) -> None:
        self.opf = opf
        self.error: Optional[str] = None
        self._lock = threading.Lock()

    def redact(self, text: str) -> Any:
        with self._lock:
            return self.opf.redact(text)


def build_redactor() -> Redactor:
    """Construct the real redactor. Imported lazily so a missing dependency
    (torch, checkpoint) surfaces as an HTTP error from a running module
    instead of an ImportError at import time — and so tests can stub it."""
    try:
        from opf import OPF
    except ImportError:
        from opf._api import OPF  # older installs export only the module path

    return Redactor(
        OPF(
            device=DEVICE,  # type: ignore[arg-type]
            output_mode="typed",
            discard_overlapping_predicted_spans=True,
        )
    )


def create_app(redactor_factory: Callable[[], Redactor] = build_redactor) -> FastAPI:
    """Build the service app. The factory parameter is the test seam; the
    default builds the real model at startup, inside the lifespan, so a
    missing checkpoint fails loudly before the first request arrives."""

    redactor: Optional[Redactor] = None

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        nonlocal redactor
        try:
            redactor = redactor_factory()
        except Exception as exc:  # noqa: BLE001 - reported over /health and /filter
            redactor = Redactor(opf=None)
            redactor.error = f"model unavailable: {exc}"
        yield

    app = FastAPI(title="Dark Pawns Privacy Filter API", lifespan=lifespan)

    @app.get("/health")
    async def health():
        if redactor is None or redactor.error is not None:
            detail = redactor.error if redactor else "starting"
            return {"status": "unhealthy", "error": detail}
        return {"status": "healthy"}

    @app.get("/categories")
    async def categories():
        return {"categories": list(WIRE_TO_OPF)}

    @app.post("/filter", response_model=FilterResponse)
    def filter_text(request: FilterRequest) -> FilterResponse:
        # Sync def on purpose: FastAPI runs it in a threadpool, keeping the
        # event loop free during CPU-bound inference.
        if redactor is None or redactor.error is not None:
            detail = redactor.error if redactor else "service is starting"
            return FilterResponse(
                filtered_text=request.text,
                detected_categories=[],
                error=detail,
            )

        wanted = request.config.categories or list(WIRE_TO_OPF)
        opf_labels = set()
        for category in wanted:
            if category not in WIRE_TO_OPF:
                return FilterResponse(
                    filtered_text=request.text,
                    detected_categories=[],
                    error=f"unknown category: {category}",
                )
            opf_labels.add(WIRE_TO_OPF[category])

        try:
            result = redactor.redact(request.text)
        except Exception as exc:  # noqa: BLE001 - the client decides the fallback
            return FilterResponse(
                filtered_text=request.text,
                detected_categories=[],
                error=f"redaction failed: {exc}",
            )

        spans = []
        detected: List[str] = []
        seen = set()
        for span in result.detected_spans:
            wire = OPF_TO_WIRE.get(getattr(span, "label", None))
            if wire is None or wire not in wanted or wire in seen:
                continue
            seen.add(wire)
            detected.append(wire)
            spans.append((span.start, span.end))

        return FilterResponse(
            filtered_text=apply_spans(
                result.text, spans, request.config.replacement, request.config.keep_length
            ),
            detected_categories=detected,
        )

    return app


def apply_spans(
    text: str, spans: List[tuple[int, int]], replacement: str, keep_length: bool
) -> str:
    """Replace the byte ranges [start, end) in `text`, leaving the rest
    untouched. Spans are non-overlapping (the model is configured to discard
    overlaps); a defensive skip guards against any that survive anyway."""
    out = []
    cursor = 0
    for start, end in sorted(spans):
        if start < cursor or end < start or end > len(text):
            continue
        out.append(text[cursor:start])
        if keep_length:
            out.append("*" * (end - start))
        else:
            out.append(replacement)
        cursor = end
    out.append(text[cursor:])
    return "".join(out)


app = create_app()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host=HOST, port=PORT)

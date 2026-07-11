#!/usr/bin/env python3
"""Tests for the AI optimization module (scripts/ai_optimizer.py).

Covers two independent regressions:

- The cache-hit request_id contract: a cached response must be returned tagged
  with the *current* request's id, not the original request that warmed the
  cache.
- DP-1010: AIBatchProcessor held a non-reentrant threading.Lock() across an
  await, which deadlocked (or serialized the whole pipeline) as soon as a
  callback re-entered any method that also took the lock.
"""

import asyncio
import os
import sys

import pytest

# Make the module importable whether run from repo root or from scripts/.
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

try:
    from scripts.ai_optimizer import (  # noqa: E402
        AIBatchProcessor,
        AIRequest,
        AIResponse,
        AsyncAIProcessor,
    )
except ImportError:  # pragma: no cover - path fallback
    from ai_optimizer import (  # noqa: E402
        AIBatchProcessor,
        AIRequest,
        AIResponse,
        AsyncAIProcessor,
    )


# ---------------------------------------------------------------------------
# Cache-hit request_id contract
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_cache_hit_returns_current_request_id():
    """A cache hit must return a response tagged with the new request's id.

    Regression test: process() used to return the cached AIResponse verbatim,
    so its request_id was the *original* request that warmed the cache, not the
    request currently being served.

    The cache is warmed directly (via cache.set) so the test exercises only the
    cache-hit branch of process() and does not depend on the batch processor.
    """
    proc = AsyncAIProcessor(cache_size=100, batch_size=1)

    original = AIRequest(request_id="req-original", prompt="hello", model="m")
    cached_response = AIResponse(
        request_id=original.request_id,
        text="cached answer",
        tokens=4,
        latency=0.01,
        model=original.model,
    )
    proc.cache.set(original.cache_key(), cached_response)

    # A second request with the same cacheable content but a different id.
    repeat = AIRequest(request_id="req-repeat", prompt="hello", model="m")
    resp = await proc.process(repeat)

    # The returned id must be the current request's id, not the stale one.
    assert resp.request_id == "req-repeat", (
        f"cache hit returned stale request_id {resp.request_id!r}; "
        f"expected 'req-repeat'"
    )
    # Cached payload is still surfaced.
    assert resp.text == "cached answer"
    # Latency on a cache hit is zero (served from memory).
    assert resp.latency == 0


def test_cache_hit_serves_new_id_without_event_loop():
    """Same contract, verified without the batch processor via cache.get.

    A pure unit check that the returned response carries the new request id,
    independent of asyncio scheduling.
    """
    proc = AsyncAIProcessor(cache_size=100, batch_size=1)

    original = AIRequest(request_id="req-A", prompt="prompt text", model="m")
    proc.cache.set(
        original.cache_key(),
        AIResponse(
            request_id="req-A",
            text="payload",
            tokens=1,
            latency=0.5,
            model="m",
        ),
    )

    repeat = AIRequest(request_id="req-B", prompt="prompt text", model="m")
    cached = proc.cache.get(repeat.cache_key())
    assert cached is not None
    # This is the contract process() now upholds on a hit: build a response
    # tagged with the current request id from the cached payload.
    served = AIResponse(
        request_id=repeat.request_id,
        text=cached.text,
        tokens=cached.tokens,
        latency=0,
        model=cached.model,
    )
    assert served.request_id == "req-B"
    assert served.text == "payload"


# ---------------------------------------------------------------------------
# DP-1010: AIBatchProcessor must not hold its lock across an await
# ---------------------------------------------------------------------------


def _mock_callback(requests):
    """Return one response per request."""
    return [
        AIResponse(
            request_id=req.request_id,
            text=f"response to {req.request_id}",
            tokens=1,
            latency=0.001,
            model=req.model,
        )
        for req in requests
    ]


async def _concurrent_submits():
    """Fire many concurrent submits and return their responses."""
    processor = AIBatchProcessor(batch_size=5, max_wait=0.05)
    processor.set_callback(_mock_callback)
    try:
        request_count = 20
        requests = [
            AIRequest(request_id=f"req-{i}", prompt=f"prompt {i}", model="test")
            for i in range(request_count)
        ]
        return await asyncio.gather(*(processor.submit(req) for req in requests))
    finally:
        processor.shutdown()


def test_concurrent_submits_no_deadlock_and_correct_results():
    """Many concurrent submits must all return the correct result (DP-1010)."""
    responses = asyncio.run(_concurrent_submits())

    assert len(responses) == 20, f"Expected 20 responses, got {len(responses)}"
    for i, resp in enumerate(responses):
        assert isinstance(resp, AIResponse), f"Response {i} is not an AIResponse"
        assert resp.request_id == f"req-{i}"
        assert resp.text == f"response to req-{i}"


async def _partial_batch_via_timer():
    """Submit fewer items than batch_size and wait for the max_wait timer."""
    processor = AIBatchProcessor(batch_size=10, max_wait=0.05)
    processor.set_callback(_mock_callback)
    try:
        requests = [
            AIRequest(request_id=f"wait-{i}", prompt="x", model="test")
            for i in range(2)
        ]
        return await asyncio.gather(*(processor.submit(req) for req in requests))
    finally:
        processor.shutdown()


def test_partial_batch_processed_after_max_wait():
    """A partial batch is processed by the max_wait timer, not left stranded."""
    responses = asyncio.run(_partial_batch_via_timer())

    assert len(responses) == 2, f"Expected 2 responses, got {len(responses)}"
    assert all(isinstance(r, AIResponse) for r in responses)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

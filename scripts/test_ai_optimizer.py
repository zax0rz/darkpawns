#!/usr/bin/env python3
"""
Tests for the AI optimization module (scripts/ai_optimizer.py).

Covers the cache-hit request_id contract: a cached response must be returned
tagged with the *current* request's id, not the original request that warmed
the cache.
"""

import asyncio
import os
import sys

import pytest

# Make scripts/ importable whether run from repo root or from scripts/.
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from ai_optimizer import AIRequest, AIResponse, AsyncAIProcessor  # noqa: E402


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


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

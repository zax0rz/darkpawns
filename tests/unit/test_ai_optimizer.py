#!/usr/bin/env python3
"""
Regression test for DP-784.

scripts/ai_optimizer.py used to call the deprecated asyncio.get_event_loop()
from inside coroutines to schedule callbacks (AIBatchProcessor.submit) and to
grab the running loop for run_in_executor (AIBatchProcessor._process_batch).
Both call sites always execute while a loop is already running, so they
should use asyncio.get_running_loop() instead.

This test is hermetic: no network, no server, no external services.
"""

import asyncio
import importlib.util
import os
import sys
import warnings

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

_SPEC = importlib.util.spec_from_file_location(
    "ai_optimizer",
    os.path.join(
        os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))),
        "scripts",
        "ai_optimizer.py",
    ),
)
ai_optimizer = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(ai_optimizer)

AIBatchProcessor = ai_optimizer.AIBatchProcessor
AIRequest = ai_optimizer.AIRequest
AIResponse = ai_optimizer.AIResponse


def _echo_callback(requests):
    """Synchronous batch callback used to drive AIBatchProcessor in tests."""
    return [
        AIResponse(
            request_id=req.request_id,
            text="ok",
            tokens=1,
            latency=0.0,
            model=req.model,
        )
        for req in requests
    ]


def test_process_batch_run_in_executor_no_get_event_loop_warning():
    """Exercise AIBatchProcessor._process_batch() directly, which calls
    asyncio.get_running_loop().run_in_executor(...). Seeding processor.batch
    directly (rather than via submit()) avoids an unrelated, pre-existing
    self-deadlock in submit() where a full batch awaits _process_batch()
    while still holding the same non-reentrant threading.Lock."""

    async def run():
        processor = AIBatchProcessor(batch_size=5, max_wait=0.01)
        processor.set_callback(_echo_callback)
        request = AIRequest(request_id="direct-1", prompt="hi")
        future = asyncio.get_running_loop().create_future()
        processor.batch.append({"request": request, "future": future})

        await processor._process_batch()
        response = await future
        assert response.request_id == "direct-1"

    with warnings.catch_warnings():
        warnings.filterwarnings(
            "error", message=r".*get_event_loop.*", category=DeprecationWarning
        )
        asyncio.run(run())


def test_submit_scheduled_batch_no_get_event_loop_warning():
    """batch_size=2 with a single submitted request forces submit() down the
    call_later() scheduling branch, exercising that get_running_loop() call
    site.

    max_wait is set long enough that the scheduled callback never actually
    fires during this test. Letting it fire would exercise a separate,
    pre-existing deadlock: _process_batch_if_ready() holds self.lock (a
    plain, non-reentrant threading.Lock) via a synchronous ``with`` block
    while awaiting _process_batch(), which tries to reacquire the same
    lock and blocks the entire event loop thread forever. That bug is
    unrelated to DP-784 and out of scope here, so this test only asserts
    that scheduling itself (the get_running_loop().call_later() call) runs
    without a get_event_loop() DeprecationWarning, then cancels cleanly.
    """

    async def run():
        processor = AIBatchProcessor(batch_size=2, max_wait=60.0)
        processor.set_callback(_echo_callback)
        request = AIRequest(request_id="scheduled-1", prompt="hello")

        task = asyncio.create_task(processor.submit(request))
        # Let submit() run synchronously up through the call_later() line;
        # it doesn't hit a real await until "return await future".
        await asyncio.sleep(0)
        if task.done():
            # Surface any exception raised while scheduling (e.g. the
            # DeprecationWarning-as-error under our filter) instead of
            # silently discarding it below.
            task.result()
        assert not task.done(), "submit() should still be waiting on its future"

        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass

    with warnings.catch_warnings():
        warnings.filterwarnings(
            "error", message=r".*get_event_loop.*", category=DeprecationWarning
        )
        asyncio.run(run())

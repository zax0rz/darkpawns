#!/usr/bin/env python3
"""
AI Optimization Module for Dark Pawns
Provides async processing, batching, and caching for AI requests.
"""

import asyncio
import json
import time
import hashlib
from typing import Dict, List, Any, Optional, Callable
from dataclasses import dataclass
from concurrent.futures import ThreadPoolExecutor
import threading
from collections import OrderedDict
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class AIRequest:
    """Represents an AI request."""
    request_id: str
    prompt: str
    model: str = "default"
    max_tokens: int = 100
    temperature: float = 0.7
    timestamp: float = None
    
    def __post_init__(self):
        if self.timestamp is None:
            self.timestamp = time.time()
    
    def cache_key(self) -> str:
        """Generate cache key for this request."""
        data = {
            "prompt": self.prompt,
            "model": self.model,
            "max_tokens": self.max_tokens,
            "temperature": self.temperature,
        }
        return hashlib.sha256(json.dumps(data, sort_keys=True).encode()).hexdigest()

@dataclass
class AIResponse:
    """Represents an AI response."""
    request_id: str
    text: str
    tokens: int
    latency: float
    model: str
    metadata: Dict[str, Any] = None
    
    def __post_init__(self):
        if self.metadata is None:
            self.metadata = {}

class AICache:
    """LRU cache for AI responses."""
    
    def __init__(self, max_size: int = 1000, ttl: int = 3600):
        self.max_size = max_size
        self.ttl = ttl
        self.cache = OrderedDict()
        self.lock = threading.RLock()
        self.hits = 0
        self.misses = 0
    
    def get(self, key: str) -> Optional[AIResponse]:
        """Get a cached response."""
        with self.lock:
            if key not in self.cache:
                self.misses += 1
                return None
            
            entry = self.cache[key]
            if time.time() - entry['timestamp'] > self.ttl:
                # Expired
                del self.cache[key]
                self.misses += 1
                return None
            
            # Move to end (most recently used)
            self.cache.move_to_end(key)
            self.hits += 1
            return entry['response']
    
    def set(self, key: str, response: AIResponse):
        """Cache a response."""
        with self.lock:
            if key in self.cache:
                # Update existing
                self.cache[key] = {
                    'response': response,
                    'timestamp': time.time()
                }
                self.cache.move_to_end(key)
            else:
                # Add new
                if len(self.cache) >= self.max_size:
                    # Remove oldest
                    self.cache.popitem(last=False)
                
                self.cache[key] = {
                    'response': response,
                    'timestamp': time.time()
                }
    
    def clear(self):
        """Clear the cache."""
        with self.lock:
            self.cache.clear()
    
    def stats(self) -> Dict[str, Any]:
        """Get cache statistics."""
        with self.lock:
            total = self.hits + self.misses
            hit_rate = self.hits / total if total > 0 else 0
            
            # Count expired entries
            now = time.time()
            expired = 0
            for entry in self.cache.values():
                if now - entry['timestamp'] > self.ttl:
                    expired += 1
            
            return {
                'size': len(self.cache),
                'max_size': self.max_size,
                'hits': self.hits,
                'misses': self.misses,
                'hit_rate': hit_rate,
                'expired': expired,
                'ttl': self.ttl,
            }

class AIBatchProcessor:
    """Batches AI requests for efficient processing."""
    
    def __init__(self, batch_size: int = 10, max_wait: float = 0.1):
        self.batch_size = batch_size
        self.max_wait = max_wait
        self.batch: List[Dict] = []
        self.lock = threading.Lock()
        self.processing = False
        self.callback = None
        self.executor = ThreadPoolExecutor(max_workers=4)
    
    def set_callback(self, callback: Callable[[List[AIRequest]], List[AIResponse]]):
        """Set the batch processing callback."""
        self.callback = callback
    
    async def submit(self, request: AIRequest) -> AIResponse:
        """Submit a request for batch processing."""
        if not self.callback:
            raise ValueError("Callback not set")
        
        future = asyncio.Future()
        
        with self.lock:
            self.batch.append({
                'request': request,
                'future': future
            })
            
            if len(self.batch) >= self.batch_size:
                await self._process_batch()
            else:
                # Schedule batch processing after max_wait
                asyncio.get_event_loop().call_later(
                    self.max_wait,
                    lambda: asyncio.create_task(self._process_batch_if_ready())
                )
        
        return await future
    
    async def _process_batch_if_ready(self):
        """Process batch if it has items."""
        with self.lock:
            if self.batch and not self.processing:
                await self._process_batch()
    
    async def _process_batch(self):
        """Process the current batch."""
        with self.lock:
            if self.processing or not self.batch:
                return
            
            self.processing = True
            batch = self.batch.copy()
            self.batch.clear()
        
        try:
            # Extract requests
            requests = [item['request'] for item in batch]
            
            # Process batch in thread pool
            loop = asyncio.get_event_loop()
            responses = await loop.run_in_executor(
                self.executor,
                lambda: self.callback(requests)
            )
            
            # Map responses to futures
            response_map = {resp.request_id: resp for resp in responses}
            
            for item in batch:
                request_id = item['request'].request_id
                if request_id in response_map:
                    item['future'].set_result(response_map[request_id])
                else:
                    item['future'].set_exception(
                        ValueError(f"No response for request {request_id}")
                    )
                    
        except Exception as e:
            logger.error(f"Batch processing error: {e}")
            for item in batch:
                item['future'].set_exception(e)
        
        finally:
            with self.lock:
                self.processing = False
    
    def shutdown(self):
        """Shutdown the batch processor."""
        self.executor.shutdown(wait=True)

class AsyncAIProcessor:
    """Async AI request processor with caching and batching."""
    
    def __init__(self, cache_size: int = 1000, batch_size: int = 10):
        self.cache = AICache(max_size=cache_size)
        self.batch_processor = AIBatchProcessor(batch_size=batch_size)
        self.batch_processor.set_callback(self._process_batch)
        self.request_count = 0
    
    async def process(self, request: AIRequest) -> AIResponse:
        """Process an AI request."""
        self.request_count += 1
        
        # Check cache first
        cache_key = request.cache_key()
        cached = self.cache.get(cache_key)
        if cached:
            logger.debug(f"Cache hit for request {request.request_id}")
            return cached
        
        # Process through batch processor
        logger.debug(f"Processing request {request.request_id}")
        response = await self.batch_processor.submit(request)
        
        # Cache the response
        self.cache.set(cache_key, response)
        
        return response
    
    def _process_batch(self, requests: List[AIRequest]) -> List[AIResponse]:
        """Process a batch of AI requests."""
        # This is a mock implementation
        # In production, this would call the actual AI API
        responses = []
        
        for request in requests:
            # Simulate AI processing
            time.sleep(0.01)  # 10ms per request
            
            response = AIResponse(
                request_id=request.request_id,
                text=f"Mock response to: {request.prompt[:50]}...",
                tokens=len(request.prompt) // 4,
                latency=0.01,
                model=request.model,
                metadata={"batch_processed": True}
            )
            responses.append(response)
        
        return responses
    
    def stats(self) -> Dict[str, Any]:
        """Get processor statistics."""
        cache_stats = self.cache.stats()
        return {
            **cache_stats,
            'total_requests': self.request_count,
            'batch_size': self.batch_processor.batch_size,
        }
    
    def shutdown(self):
        """Shutdown the processor."""
        self.batch_processor.shutdown()

class WebSocketOptimizer:
    """Optimizes WebSocket communication."""
    
    def __init__(self, max_buffer_size: int = 256):
        self.max_buffer_size = max_buffer_size
        self.sessions: Dict[str, asyncio.Queue] = {}
        self.lock = threading.RLock()
        self.metrics = {
            'messages_sent': 0,
            'messages_dropped': 0,
            'broadcasts': 0,
        }
    
    def register_session(self, session_id: str, queue: asyncio.Queue):
        """Register a WebSocket session."""
        with self.lock:
            self.sessions[session_id] = queue
    
    def unregister_session(self, session_id: str):
        """Unregister a WebSocket session."""
        with self.lock:
            if session_id in self.sessions:
                del self.sessions[session_id]
    
    async def send_to_session(self, session_id: str, message: Dict[str, Any]):
        """Send a message to a specific session."""
        with self.lock:
            if session_id not in self.sessions:
                logger.warning(f"Session {session_id} not found")
                return False
            
            queue = self.sessions[session_id]
            
            if queue.qsize() >= self.max_buffer_size:
                self.metrics['messages_dropped'] += 1
                logger.warning(f"Buffer full for session {session_id}, dropping message")
                return False
            
            await queue.put(json.dumps(message))
            self.metrics['messages_sent'] += 1
            return True
    
    async def broadcast(self, message: Dict[str, Any], exclude: List[str] = None):
        """Broadcast a message to all sessions."""
        if exclude is None:
            exclude = []
        
        with self.lock:
            session_ids = list(self.sessions.keys())
        
        success_count = 0
        for session_id in session_ids:
            if session_id in exclude:
                continue
            
            if await self.send_to_session(session_id, message):
                success_count += 1
        
        self.metrics['broadcasts'] += 1
        return success_count
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get optimizer metrics."""
        with self.lock:
            return {
                **self.metrics,
                'active_sessions': len(self.sessions),
                'max_buffer_size': self.max_buffer_size,
            }

# Example usage
async def example_usage():
    """Example of using the AI optimizer."""
    # Create processor
    processor = AsyncAIProcessor(cache_size=100, batch_size=5)
    
    # Create some requests
    requests = [
        AIRequest(
            request_id=f"req-{i}",
            prompt=f"What is the meaning of life? Request {i}",
            model="example-model"
        )
        for i in range(20)
    ]
    
    # Process requests concurrently
    tasks = [processor.process(req) for req in requests]
    responses = await asyncio.gather(*tasks, return_exceptions=True)
    
    # Print results
    for i, (req, resp) in enumerate(zip(requests, responses)):
        if isinstance(resp, Exception):
            print(f"Request {req.request_id} failed: {resp}")
        else:
            print(f"Request {req.request_id}: {resp.text[:50]}...")
    
    # Print stats
    stats = processor.stats()
    print(f"\nProcessor stats: {stats}")
    
    # Cleanup
    processor.shutdown()

if __name__ == "__main__":
    # Run example
    asyncio.run(example_usage())
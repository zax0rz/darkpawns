#!/usr/bin/env python3
"""
Main emotion tagging pipeline for Dark Pawns narrative memory.

Combines rule-based and LLM classifiers for robust emotional tagging.
"""

import json
from typing import Dict, List, Optional, Any
from datetime import datetime
import time

# Import classifiers
try:
    from emotion_classifier import RuleBasedEmotionClassifier
    from emotion_llm_classifier import LLMEmotionClassifier
except ImportError:
    # Create minimal versions for testing
    class RuleBasedEmotionClassifier:
        def classify(self, text, context=None):
            return {"category": "neutral", "intensity": 1, "primary_emotions": [], 
                   "confidence": 0.3, "method": "mock"}
    
    class LLMEmotionClassifier:
        def __init__(self, model="minimax-m2.7", use_fallback=True):
            self.model = model
            self.use_fallback = use_fallback
        
        def classify(self, text, context=None):
            return {"category": "neutral", "intensity": 1, "primary_emotions": [], 
                   "confidence": 0.3, "method": f"llm_{self.model}"}


class EmotionTagger:
    """Main emotion tagging pipeline"""
    
    def __init__(self, use_llm: bool = True, llm_model: str = "minimax-m2.7",
                 llm_threshold: float = 0.7, cache_results: bool = True):
        """
        Initialize emotion tagger.
        
        Args:
            use_llm: Whether to use LLM classifier for low-confidence cases
            llm_model: LLM model to use
            llm_threshold: Confidence threshold below which to use LLM (0.0-1.0)
            cache_results: Whether to cache classification results
        """
        self.use_llm = use_llm
        self.llm_threshold = llm_threshold
        self.cache_results = cache_results
        
        # Initialize classifiers
        self.rule_classifier = RuleBasedEmotionClassifier()
        
        if use_llm:
            self.llm_classifier = LLMEmotionClassifier(model=llm_model, use_fallback=True)
        else:
            self.llm_classifier = None
        
        # Result cache (text_hash -> result)
        self.cache = {}
        
        # Statistics
        self.stats = {
            "total_classifications": 0,
            "rule_based_classifications": 0,
            "llm_classifications": 0,
            "fallback_classifications": 0,
            "avg_confidence": 0.0,
            "category_distribution": {"positive": 0, "negative": 0, "neutral": 0}
        }
    
    def _get_text_hash(self, text: str, context: Optional[Dict] = None) -> str:
        """Create hash for caching"""
        import hashlib
        content = text + json.dumps(context or {}, sort_keys=True)
        return hashlib.md5(content.encode()).hexdigest()
    
    def tag_memory(self, text: str, context: Optional[Dict] = None) -> Dict:
        """Tag a memory with emotional valence"""
        
        # Check cache
        text_hash = self._get_text_hash(text, context)
        if self.cache_results and text_hash in self.cache:
            cached_result = self.cache[text_hash]
            cached_result["cached"] = True
            return cached_result
        
        # First pass: rule-based
        rule_result = self.rule_classifier.classify(text, context)
        self.stats["rule_based_classifications"] += 1
        
        # Determine if we need LLM refinement
        use_llm = False
        if self.use_llm and self.llm_classifier:
            # Use LLM if confidence is low
            if rule_result["confidence"] < self.llm_threshold:
                use_llm = True
            # Use LLM for high-stakes events regardless of confidence
            elif context and context.get("valence", 0) in [-3, 3]:  # Extreme valence
                use_llm = True
            # Use LLM for social events
            elif context and context.get("event_type") == "player_encounter":
                use_llm = True
        
        if use_llm:
            llm_result = self.llm_classifier.classify(text, context)
            self.stats["llm_classifications"] += 1
            
            # Check if LLM used fallback
            if "fallback" in llm_result.get("method", ""):
                self.stats["fallback_classifications"] += 1
            
            # Prefer LLM if confidence is higher or it's an extreme case
            llm_confidence = llm_result.get("confidence", 0)
            rule_confidence = rule_result.get("confidence", 0)
            
            if (llm_confidence > rule_confidence or 
                context and abs(context.get("valence", 0)) >= 2):
                final_result = llm_result
            else:
                final_result = rule_result
                final_result["llm_considered"] = True
                final_result["llm_confidence"] = llm_confidence
        else:
            final_result = rule_result
        
        # Update statistics
        self.stats["total_classifications"] += 1
        category = final_result["category"]
        if category in self.stats["category_distribution"]:
            self.stats["category_distribution"][category] += 1
        
        # Update average confidence
        total = self.stats["total_classifications"]
        old_avg = self.stats["avg_confidence"]
        self.stats["avg_confidence"] = old_avg + (final_result["confidence"] - old_avg) / total
        
        # Add timestamp and context info
        final_result["timestamp"] = datetime.now().isoformat()
        if context:
            final_result["context_summary"] = {
                "event_type": context.get("event_type"),
                "valence": context.get("valence"),
                "related_entity": context.get("related_entity")
            }
        
        # Cache result
        if self.cache_results:
            self.cache[text_hash] = final_result.copy()
            final_result["cached"] = False
        
        return final_result
    
    def batch_tag(self, memories: List[Dict], batch_size: int = 10) -> List[Dict]:
        """Tag multiple memories with optional batching"""
        results = []
        
        for i in range(0, len(memories), batch_size):
            batch = memories[i:i + batch_size]
            print(f"Processing batch {i//batch_size + 1}/{(len(memories) + batch_size - 1)//batch_size}")
            
            for memory in batch:
                text = memory.get("text", "")
                context = memory.get("context", {})
                
                tags = self.tag_memory(text, context)
                
                results.append({
                    "memory_id": memory.get("id"),
                    "text": text,
                    "tags": tags,
                    "processed_at": datetime.now().isoformat()
                })
            
            # Small delay between batches
            if i + batch_size < len(memories):
                time.sleep(1)
        
        return results
    
    def get_statistics(self) -> Dict:
        """Get classification statistics"""
        stats = self.stats.copy()
        
        # Calculate percentages
        total = stats["total_classifications"]
        if total > 0:
            stats["rule_based_percentage"] = stats["rule_based_classifications"] / total * 100
            stats["llm_percentage"] = stats["llm_classifications"] / total * 100
            stats["fallback_percentage"] = stats["fallback_classifications"] / total * 100
            
            # Category percentages
            for category in stats["category_distribution"]:
                count = stats["category_distribution"][category]
                stats["category_distribution"][f"{category}_percentage"] = count / total * 100
        
        return stats
    
    def clear_cache(self):
        """Clear the result cache"""
        self.cache.clear()
    
    def save_cache(self, filepath: str):
        """Save cache to file"""
        with open(filepath, 'w') as f:
            json.dump({
                "cache": self.cache,
                "stats": self.stats,
                "timestamp": datetime.now().isoformat()
            }, f, indent=2)
    
    def load_cache(self, filepath: str):
        """Load cache from file"""
        try:
            with open(filepath, 'r') as f:
                data = json.load(f)
                self.cache = data.get("cache", {})
                self.stats = data.get("stats", self.stats.copy())
        except FileNotFoundError:
            print(f"Cache file not found: {filepath}")
        except json.JSONDecodeError as e:
            print(f"Error loading cache: {e}")


# Database integration functions
def update_postgres_memory(db_conn, memory_id: int, tags: Dict):
    """Update Postgres memory with emotional tags"""
    try:
        import psycopg2
        
        query = """
        UPDATE agent_narrative_memory 
        SET emotion_category = %s,
            emotion_intensity = %s,
            primary_emotions = %s,
            emotion_confidence = %s,
            updated_at = NOW()
        WHERE id = %s
        """
        
        db_conn.execute(query, (
            tags["category"],
            tags["intensity"],
            json.dumps(tags.get("primary_emotions", [])),
            tags["confidence"],
            memory_id
        ))
        
        return True
        
    except Exception as e:
        print(f"Error updating Postgres: {e}")
        return False


def create_qdrant_metadata(tags: Dict, additional_metadata: Optional[Dict] = None) -> Dict:
    """Create Qdrant metadata structure with emotional tags"""
    metadata = additional_metadata or {}
    
    metadata["emotional_tags"] = {
        "category": tags["category"],
        "intensity": tags["intensity"],
        "primary_emotions": tags.get("primary_emotions", []),
        "confidence": tags["confidence"],
        "method": tags.get("method", "unknown"),
        "timestamp": tags.get("timestamp", datetime.now().isoformat())
    }
    
    return metadata


def test_tagger():
    """Test the emotion tagger"""
    
    tagger = EmotionTagger(use_llm=False)  # Test without LLM for speed
    
    test_memories = [
        {
            "id": 1,
            "text": "Killed the dragon and took its treasure. Feeling invincible!",
            "context": {"event_type": "mob_kill", "valence": 3, "related_entity": "dragon"}
        },
        {
            "id": 2,
            "text": "Died to a rat in the sewers. Embarrassing.",
            "context": {"event_type": "mob_death", "valence": -3, "related_entity": "rat"}
        },
        {
            "id": 3,
            "text": "Moved from room 100 to room 101.",
            "context": {"event_type": "room_visit", "valence": 0}
        },
        {
            "id": 4,
            "text": "Barely survived the orc attack. Almost died but managed to flee.",
            "context": {"event_type": "combat", "valence": -2, "related_entity": "orc"}
        },
        {
            "id": 5,
            "text": "Found an amazing magical sword! This is fantastic!",
            "context": {"event_type": "item_loot", "valence": 2, "related_entity": "magical sword"}
        }
    ]
    
    print("Testing EmotionTagger:")
    print("=" * 80)
    
    # Test single classification
    test_text = "Killed the dragon and took its treasure. Feeling invincible!"
    test_context = {"event_type": "mob_kill", "valence": 3}
    
    result = tagger.tag_memory(test_text, test_context)
    print(f"Single classification:")
    print(f"Text: {test_text}")
    print(f"Result: {json.dumps(result, indent=2)}")
    print("-" * 80)
    
    # Test batch classification
    print(f"Batch classification ({len(test_memories)} memories):")
    results = tagger.batch_tag(test_memories, batch_size=2)
    
    for i, result in enumerate(results):
        print(f"\nMemory {i+1}:")
        print(f"Text: {result['text'][:50]}...")
        tags = result['tags']
        print(f"Category: {tags['category']} (intensity: {tags['intensity']})")
        print(f"Primary emotions: {tags.get('primary_emotions', [])}")
        print(f"Confidence: {tags['confidence']}")
        print(f"Method: {tags.get('method', 'unknown')}")
    
    print("\n" + "=" * 80)
    print("Statistics:")
    stats = tagger.get_statistics()
    print(json.dumps(stats, indent=2))
    
    # Test Qdrant metadata creation
    print("\n" + "=" * 80)
    print("Qdrant metadata example:")
    metadata = create_qdrant_metadata(result['tags'], {"event_type": "mob_kill"})
    print(json.dumps(metadata, indent=2))


if __name__ == "__main__":
    test_tagger()
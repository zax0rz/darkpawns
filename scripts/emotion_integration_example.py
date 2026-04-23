#!/usr/bin/env python3
"""
Emotion tagging integration example for Dark Pawns.

Shows how to integrate emotional valence tagging with:
1. Existing dp_brenda.py memory system
2. Postgres narrative memory
3. Real-time event processing
"""

import json
import os
import sys
from datetime import datetime
from typing import Dict, Optional

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

try:
    from scripts.emotion_tagger import EmotionTagger, create_qdrant_metadata
except ImportError:
    try:
        from emotion_tagger import EmotionTagger, create_qdrant_metadata
    except ImportError as e:
        print(f"Import error: {e}")
        sys.exit(1)


# Mock classes for demonstration
class MockMemorySystem:
    """Mock memory system similar to dp_brenda.py"""
    
    def __init__(self):
        self.tagger = EmotionTagger(use_llm=True, llm_threshold=0.6)
        self.memories = []
    
    def add_memory(self, text: str, metadata: Optional[Dict] = None, 
                   event_context: Optional[Dict] = None):
        """Add memory with emotional tagging"""
        
        # Tag memory with emotion
        tags = self.tagger.tag_memory(text, event_context)
        
        # Create enhanced metadata
        enhanced_metadata = metadata or {}
        enhanced_metadata.update(create_qdrant_metadata(tags, metadata))
        
        # Store memory
        memory = {
            "text": text,
            "metadata": enhanced_metadata,
            "tags": tags,
            "timestamp": datetime.now().isoformat()
        }
        
        self.memories.append(memory)
        
        print(f"Added memory with emotional tags: {tags['category']} ({tags['intensity']})")
        return memory
    
    def query_by_emotion(self, category: Optional[str] = None, 
                         min_intensity: int = 1) -> list:
        """Query memories by emotional criteria"""
        results = []
        
        for memory in self.memories:
            tags = memory["tags"]
            
            if category and tags["category"] != category:
                continue
            
            if tags["intensity"] < min_intensity:
                continue
            
            results.append(memory)
        
        # Sort by intensity (highest first)
        results.sort(key=lambda x: x["tags"]["intensity"], reverse=True)
        return results
    
    def get_emotional_stats(self) -> Dict:
        """Get emotional statistics"""
        stats = self.tagger.get_statistics()
        
        # Add memory-specific stats
        category_counts = {"positive": 0, "negative": 0, "neutral": 0}
        for memory in self.memories:
            category = memory["tags"]["category"]
            if category in category_counts:
                category_counts[category] += 1
        
        stats["memory_category_counts"] = category_counts
        return stats


class MockGameEvent:
    """Mock game event similar to Dark Pawns events"""
    
    @staticmethod
    def create_mob_kill_event(mob_name: str, room_name: str, difficulty: str = "medium"):
        """Create a mob kill event"""
        return {
            "type": "mob_kill",
            "mob": mob_name,
            "room": room_name,
            "difficulty": difficulty,
            "timestamp": datetime.now().isoformat(),
            "valence": 3 if difficulty == "hard" else 2 if difficulty == "medium" else 1
        }
    
    @staticmethod
    def create_mob_death_event(mob_name: str, room_name: str, embarrassing: bool = False):
        """Create a mob death event"""
        return {
            "type": "mob_death",
            "mob": mob_name,
            "room": room_name,
            "embarrassing": embarrassing,
            "timestamp": datetime.now().isoformat(),
            "valence": -3 if embarrassing else -2
        }
    
    @staticmethod
    def create_item_loot_event(item_name: str, quality: str = "common"):
        """Create an item loot event"""
        valence_map = {"common": 1, "uncommon": 2, "rare": 3, "epic": 4, "legendary": 5}
        return {
            "type": "item_loot",
            "item": item_name,
            "quality": quality,
            "timestamp": datetime.now().isoformat(),
            "valence": valence_map.get(quality, 1)
        }


def demonstrate_integration():
    """Demonstrate emotion tagging integration"""
    print("Dark Pawns Emotion Tagging Integration Demo")
    print("=" * 80)
    
    # Initialize systems
    memory_system = MockMemorySystem()
    game_event = MockGameEvent()
    
    print("\n1. Processing game events with emotional tagging:")
    print("-" * 40)
    
    # Process various game events
    events = [
        {
            "event": game_event.create_mob_kill_event("dragon", "Dragon's Lair", "hard"),
            "text": "Slayed the mighty dragon in its lair! Epic victory!"
        },
        {
            "event": game_event.create_mob_death_event("rat", "Sewers", embarrassing=True),
            "text": "Died to a sewer rat. How embarrassing..."
        },
        {
            "event": game_event.create_item_loot_event("Sword of Destiny", "legendary"),
            "text": "Found the legendary Sword of Destiny! This changes everything!"
        },
        {
            "event": game_event.create_mob_kill_event("goblin", "Forest", "easy"),
            "text": "Dispatched a goblin in the forest."
        },
        {
            "event": game_event.create_mob_death_event("orc warlord", "Throne Room"),
            "text": "Fell to the orc warlord. It was a tough fight."
        }
    ]
    
    for event_info in events:
        event = event_info["event"]
        text = event_info["text"]
        
        # Create context for emotion tagging
        context = {
            "event_type": event["type"],
            "valence": event.get("valence", 0),
            "related_entity": event.get("mob") or event.get("item"),
            "room_name": event.get("room", ""),
            "additional_info": {
                "difficulty": event.get("difficulty"),
                "quality": event.get("quality"),
                "embarrassing": event.get("embarrassing", False)
            }
        }
        
        # Create metadata for Qdrant
        metadata = {
            "event_type": event["type"],
            "timestamp": event["timestamp"],
            "game_data": {
                "mob": event.get("mob"),
                "item": event.get("item"),
                "room": event.get("room"),
                "difficulty": event.get("difficulty"),
                "quality": event.get("quality")
            }
        }
        
        # Add memory with emotional tagging
        memory = memory_system.add_memory(text, metadata, context)
        
        # Print details
        tags = memory["tags"]
        print(f"  Event: {event['type']} - {event.get('mob') or event.get('item')}")
        print(f"    Emotion: {tags['category']} (intensity: {tags['intensity']})")
        print(f"    Primary emotions: {tags.get('primary_emotions', [])}")
        print(f"    Confidence: {tags['confidence']}")
        print(f"    Method: {tags.get('method', 'unknown')}")
        print()
    
    print("\n2. Querying memories by emotion:")
    print("-" * 40)
    
    # Query for intense positive memories
    print("Intense positive memories (intensity >= 4):")
    positive_memories = memory_system.query_by_emotion(category="positive", min_intensity=4)
    
    for i, memory in enumerate(positive_memories, 1):
        tags = memory["tags"]
        print(f"  {i}. {memory['text'][:60]}...")
        print(f"     Intensity: {tags['intensity']}, Confidence: {tags['confidence']}")
    
    # Query for negative memories
    print("\nNegative memories:")
    negative_memories = memory_system.query_by_emotion(category="negative")
    
    for i, memory in enumerate(negative_memories, 1):
        tags = memory["tags"]
        print(f"  {i}. {memory['text'][:60]}...")
        print(f"     Intensity: {tags['intensity']}, Primary emotions: {tags.get('primary_emotions', [])}")
    
    print("\n3. Emotional statistics:")
    print("-" * 40)
    
    stats = memory_system.get_emotional_stats()
    
    print("Tagger statistics:")
    print(f"  Total classifications: {stats['total_classifications']}")
    print(f"  Rule-based: {stats['rule_based_classifications']} ({stats.get('rule_based_percentage', 0):.1f}%)")
    print(f"  LLM: {stats['llm_classifications']} ({stats.get('llm_percentage', 0):.1f}%)")
    print(f"  Average confidence: {stats['avg_confidence']:.3f}")
    
    print("\nMemory category distribution:")
    category_counts = stats.get('memory_category_counts', {})
    total_memories = sum(category_counts.values())
    
    for category, count in category_counts.items():
        percentage = count / total_memories * 100 if total_memories > 0 else 0
        print(f"  {category}: {count} ({percentage:.1f}%)")
    
    print("\n4. Example Qdrant metadata:")
    print("-" * 40)
    
    # Show example metadata for the first memory
    if memory_system.memories:
        example_memory = memory_system.memories[0]
        print("First memory metadata structure:")
        print(json.dumps(example_memory["metadata"], indent=2))
    
    print("\n5. Integration with existing dp_brenda.py:")
    print("-" * 40)
    
    print("""
To integrate with existing dp_brenda.py:

1. Import EmotionTagger in dp_brenda.py:
   ```python
   from emotion_tagger import EmotionTagger, create_qdrant_metadata
   ```

2. Initialize in BrendaMemory class:
   ```python
   class BrendaMemory:
       def __init__(self):
           # ... existing code ...
           self.emotion_tagger = EmotionTagger(use_llm=True)
   ```

3. Modify add_async method:
   ```python
   def add_with_emotion(self, text: str, metadata: Optional[dict] = None, 
                        event_context: Optional[dict] = None):
       # Tag emotion
       tags = self.emotion_tagger.tag_memory(text, event_context)
       
       # Enhance metadata
       enhanced_metadata = dict(metadata or {})
       enhanced_metadata.update(create_qdrant_metadata(tags, metadata))
       
       # Store memory
       self.memory.add(text, user_id="brenda69", metadata=enhanced_metadata)
   ```

4. Use in event processing:
   ```python
   async def process_event(self, event: dict):
       # ... existing code ...
       
       if event["type"] in ["mob_kill", "mob_death", "player_encounter"]:
           memory_text = self._create_memory_text(event)
           context = {
               "event_type": event["type"],
               "valence": event.get("valence", 0),
               "related_entity": event.get("mob") or event.get("player")
           }
           
           self.memory.add_with_emotion(
               memory_text,
               metadata={"event": event["type"], "timestamp": event.get("timestamp")},
               event_context=context
           )
   ```
    """)
    
    print("\n" + "=" * 80)
    print("Integration demonstration complete!")
    print("\nKey benefits:")
    print("1. Enhanced memory retrieval by emotional content")
    print("2. Better agent personality expression")
    print("3. Research data on emotional learning in game agents")
    print("4. Compatibility with existing Postgres + Qdrant architecture")


if __name__ == "__main__":
    demonstrate_integration()
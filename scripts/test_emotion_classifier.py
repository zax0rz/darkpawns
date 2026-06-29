#!/usr/bin/env python3
"""
Test script for emotion classifier components.

Tests rule-based classifier, LLM classifier, and main tagger pipeline.
"""

import sys
import os
import json
from datetime import datetime

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# Import classifiers
try:
    from scripts.emotion_classifier import RuleBasedEmotionClassifier
    from scripts.emotion_llm_classifier import LLMEmotionClassifier
    from scripts.emotion_tagger import EmotionTagger
except ImportError:
    try:
        from emotion_classifier import RuleBasedEmotionClassifier
        from emotion_llm_classifier import LLMEmotionClassifier
        from emotion_tagger import EmotionTagger
    except ImportError as e:
        print(f"Import error: {e}")
        print("Make sure all classifier files are in the same directory.")
        sys.exit(1)


def test_rule_based_classifier():
    """Test rule-based emotion classifier"""
    print("Testing RuleBasedEmotionClassifier")
    print("=" * 80)
    
    classifier = RuleBasedEmotionClassifier()
    
    test_cases = [
        {
            "text": "Killed the dragon and took its treasure. Feeling invincible!",
            "expected_category": "positive",
            "expected_min_intensity": 3
        },
        {
            "text": "Died to a rat in the sewers. Embarrassing and frustrating.",
            "expected_category": "negative",
            "expected_min_intensity": 3
        },
        {
            "text": "Moved from room 100 to room 101.",
            "expected_category": "neutral",
            "expected_min_intensity": 1
        },
        {
            "text": "Barely survived the orc attack. Almost died but managed to flee.",
            "expected_category": "negative",
            "expected_min_intensity": 2
        },
        {
            "text": "Found an amazing magical sword! This is fantastic!",
            "expected_category": "positive",
            "expected_min_intensity": 4
        },
        {
            "text": "The goblin ambush was terrifying. I was so scared.",
            "expected_category": "negative",
            "expected_min_intensity": 3
        },
        {
            "text": "Not happy with the loot from that chest. Disappointed.",
            "expected_category": "negative",
            "expected_min_intensity": 2
        },
        {
            "text": "Absolutely destroyed the troll king. Total victory!",
            "expected_category": "positive",
            "expected_min_intensity": 5
        }
    ]
    
    passed = 0
    total = len(test_cases)
    
    for i, test in enumerate(test_cases, 1):
        result = classifier.classify(test["text"])

        # Assert result structure and values
        assert isinstance(result, dict), f"Expected dict, got {type(result)}"
        assert "category" in result, f"Test {i}: missing 'category' in result"
        assert "intensity" in result
        assert "confidence" in result
        assert result['category'] == test['expected_category'], \
            f"Test {i}: {test['text'][:30]}... expected {test['expected_category']}, got {result['category']}"
        assert result['intensity'] >= test['expected_min_intensity'], \
            f"Test {i}: expected intensity >= {test['expected_min_intensity']}, got {result['intensity']}"

        passed += 1
        print(f"  Test {i}: {str(test['text'])[:40]:40} ✓ {result['category']} (intensity={result['intensity']})")

    print(f"\nRule-based classifier: {passed}/{total} tests passed")
    return passed == total


def test_llm_classifier_mock():
    """Test LLM classifier with mock data (no actual API calls)"""
    print("\n\nTesting LLMEmotionClassifier (mock mode)")
    print("=" * 80)

    # Create classifier with fallback
    classifier = LLMEmotionClassifier(model="minimax-m2.7", use_fallback=True)

    test_text = "Killed the dragon and took its treasure. Feeling invincible!"
    context = {"event_type": "mob_kill", "valence": 3}

    result = classifier.classify(test_text, context)

    print(f"Text: {test_text}")
    print(f"Result: {json.dumps(result, indent=2)}")

    # Assert valid result structure
    required_fields = ['category', 'intensity', 'confidence', 'method']
    for field in required_fields:
        assert field in result, f"LLM classifier result missing required field '{field}": got keys {list(result.keys())}"'
    
    print("✓ LLM classifier returned valid result structure")
    return True


def test_emotion_tagger():
    """Test main emotion tagger pipeline"""
    print("\n\nTesting EmotionTagger")
    print("=" * 80)

    # Test without LLM for speed
    tagger = EmotionTagger(use_llm=False, cache_results=True)

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
        }
    ]

    # Single classification
    test_text = "Found an amazing magical sword! This is fantastic!"
    test_context = {"event_type": "item_loot", "valence": 2}

    result = tagger.tag_memory(test_text, test_context)
    assert isinstance(result, dict), f"Expected dict, got {type(result)}"
    assert "category" in result, f"Result missing 'category', keys: {list(result.keys())}"
    assert "intensity" in result
    assert "confidence" in result
    assert isinstance(result["confidence"], (int, float)), \
        f"confidence should be numeric, got {type(result['confidence'])}"
    assert 0 <= result["confidence"] <= 1, \
        f"confidence {result['confidence']} out of range [0,1]"
    print(f"  [PASS] Single: {result['category']} (intensity={result['intensity']}, "
          f"confidence={result['confidence']}, method={result.get('method', 'unknown')})")

    # Batch classification
    results = tagger.batch_tag(test_memories, batch_size=2)
    assert len(results) == len(test_memories), \
        f"Expected {len(test_memories)} results, got {len(results)}"
    for r in results:
        assert "text" in r
        assert "tags" in r
        tags = r["tags"]
        assert "category" in tags
        assert "intensity" in tags
        assert "confidence" in tags
    print(f"  [PASS] Batch: {len(results)} memories tagged")

    # Statistics
    stats = tagger.get_statistics()
    assert isinstance(stats, dict), f"Expected dict, got {type(stats)}"
    print(f"  [PASS] Statistics: {json.dumps(stats)}")

    # Cache test
    cached_result = tagger.tag_memory(test_text, test_context)
    assert "cached" in cached_result
    assert cached_result["cached"] is True
    print(f"  [PASS] Cache hit: {cached_result.get('cached')}")

    # Clear cache and verify
    tagger.clear_cache()
    uncached_result = tagger.tag_memory(test_text, test_context)
    # After clearing, cached should be False or field absent
    assert not uncached_result.get("cached", False), \
        "Expected uncached result after clear_cache"
    print(f"  [PASS] Cache cleared: cached={uncached_result.get('cached')}")

    return True


def test_qdrant_metadata():
    """Test Qdrant metadata creation"""
    print("\n\nTesting Qdrant metadata creation")
    print("=" * 80)
    
    from scripts.emotion_tagger import create_qdrant_metadata
    
    # Create sample tags
    tags = {
        "category": "positive",
        "intensity": 4,
        "primary_emotions": ["joy", "trust"],
        "confidence": 0.92,
        "method": "rule_based",
        "timestamp": datetime.now().isoformat()
    }
    
    # Create metadata
    additional_metadata = {
        "event_type": "mob_kill",
        "room": "Dragon's Lair",
        "related_entity": "dragon"
    }
    
    metadata = create_qdrant_metadata(tags, additional_metadata)
    
    print("Generated Qdrant metadata:")
    print(json.dumps(metadata, indent=2))
    
    # Check structure
    has_emotional_tags = "emotional_tags" in metadata
    emotional_tags = metadata.get("emotional_tags", {})
    has_required_fields = all(field in emotional_tags for field in 
                             ["category", "intensity", "confidence", "method"])
    
    if has_emotional_tags and has_required_fields:
        print("✓ Qdrant metadata has correct structure")
        return True
    else:
        print("✗ Qdrant metadata missing required fields")
        return False


def test_sql_migration():
    """Test SQL migration syntax"""
    print("\n\nTesting SQL migration syntax")
    print("=" * 80)
    
    migration_file = "scripts/migrations/001_add_emotional_tags.sql"
    
    if os.path.exists(migration_file):
        print(f"✓ Migration file exists: {migration_file}")
        
        # Check file size
        file_size = os.path.getsize(migration_file)
        print(f"  File size: {file_size} bytes")
        
        # Check for key SQL statements
        with open(migration_file, 'r') as f:
            content = f.read()
        
        required_statements = [
            "ALTER TABLE agent_narrative_memory",
            "ADD COLUMN emotion_category",
            "ADD COLUMN emotion_intensity",
            "CREATE INDEX",
            "CREATE OR REPLACE VIEW"
        ]
        
        missing = []
        for statement in required_statements:
            if statement not in content:
                missing.append(statement)
        
        if not missing:
            print("✓ All required SQL statements found")
            return True
        else:
            print(f"✗ Missing SQL statements: {missing}")
            return False
    else:
        print(f"✗ Migration file not found: {migration_file}")
        return False


def run_all_tests():
    """Run all tests"""
    print("Running emotion classifier tests")
    print("=" * 80)
    
    test_results = []
    
    # Run tests
    test_results.append(("Rule-based classifier", test_rule_based_classifier()))
    test_results.append(("LLM classifier (mock)", test_llm_classifier_mock()))
    test_results.append(("Emotion tagger", test_emotion_tagger()))
    test_results.append(("Qdrant metadata", test_qdrant_metadata()))
    test_results.append(("SQL migration", test_sql_migration()))
    
    # Print summary
    print("\n\nTEST SUMMARY")
    print("=" * 80)
    
    passed = 0
    total = len(test_results)
    
    for test_name, result in test_results:
        status = "✓ PASS" if result else "✗ FAIL"
        print(f"{test_name:30} {status}")
        if result:
            passed += 1
    
    print(f"\nTotal: {passed}/{total} tests passed ({passed/total*100:.1f}%)")
    
    if passed == total:
        print("\n✓ All tests passed!")
        return True
    else:
        print("\n✗ Some tests failed")
        return False


if __name__ == "__main__":
    success = run_all_tests()
    sys.exit(0 if success else 1)
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
        
        print(f"\nTest {i}: {test['text'][:50]}...")
        print(f"  Expected: {test['expected_category']} (min intensity: {test['expected_min_intensity']})")
        print(f"  Got: {result['category']} (intensity: {result['intensity']})")
        print(f"  Confidence: {result['confidence']}")
        print(f"  Primary emotions: {result['primary_emotions']}")
        
        # Check results
        category_correct = result['category'] == test['expected_category']
        intensity_correct = result['intensity'] >= test['expected_min_intensity']
        
        if category_correct and intensity_correct:
            print("  ✓ PASS")
            passed += 1
        else:
            print("  ✗ FAIL")
            if not category_correct:
                print(f"    Category mismatch: expected {test['expected_category']}, got {result['category']}")
            if not intensity_correct:
                print(f"    Intensity too low: expected at least {test['expected_min_intensity']}, got {result['intensity']}")
    
    print(f"\nRule-based classifier: {passed}/{total} tests passed ({passed/total*100:.1f}%)")
    return passed == total


def test_llm_classifier_mock():
    """Test LLM classifier with mock data (no actual API calls)"""
    print("\n\nTesting LLMEmotionClassifier (mock mode)")
    print("=" * 80)
    
    # Create classifier with fallback
    classifier = LLMEmotionClassifier(model="minimax-m2.7", use_fallback=True)
    
    # Test with simple text
    test_text = "Killed the dragon and took its treasure. Feeling invincible!"
    context = {"event_type": "mob_kill", "valence": 3}
    
    result = classifier.classify(test_text, context)
    
    print(f"Text: {test_text}")
    print(f"Result: {json.dumps(result, indent=2)}")
    
    # Check that we got a valid result
    required_fields = ['category', 'intensity', 'confidence', 'method']
    has_required = all(field in result for field in required_fields)
    
    if has_required:
        print("✓ LLM classifier returned valid result structure")
        return True
    else:
        print("✗ LLM classifier missing required fields")
        return False


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
    
    print("Testing single classification:")
    test_text = "Found an amazing magical sword! This is fantastic!"
    test_context = {"event_type": "item_loot", "valence": 2}
    
    result = tagger.tag_memory(test_text, test_context)
    print(f"Text: {test_text}")
    print(f"Result: category={result['category']}, intensity={result['intensity']}, "
          f"confidence={result['confidence']}, method={result.get('method', 'unknown')}")
    
    print("\nTesting batch classification:")
    results = tagger.batch_tag(test_memories, batch_size=2)
    
    for i, result in enumerate(results, 1):
        print(f"\nMemory {i}:")
        print(f"  Text: {result['text'][:50]}...")
        tags = result['tags']
        print(f"  Category: {tags['category']}")
        print(f"  Intensity: {tags['intensity']}")
        print(f"  Confidence: {tags['confidence']}")
    
    print("\nTagger statistics:")
    stats = tagger.get_statistics()
    print(json.dumps(stats, indent=2))
    
    # Test cache
    print("\nTesting cache:")
    cached_result = tagger.tag_memory(test_text, test_context)
    print(f"Cached result has 'cached' field: {'cached' in cached_result}")
    
    # Clear cache and test again
    tagger.clear_cache()
    uncached_result = tagger.tag_memory(test_text, test_context)
    print(f"After clearing cache, 'cached' field: {'cached' in uncached_result}")
    
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
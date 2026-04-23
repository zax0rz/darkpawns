#!/usr/bin/env python3
"""
LLM-based emotion classifier for Dark Pawns narrative memory.

Uses MiniMax M2.7 or GLM-5.1 for advanced emotion classification.
"""

import os
import json
import re
from typing import Dict, Optional, Any
from datetime import datetime

try:
    from litellm import completion
    LITELLM_AVAILABLE = True
except ImportError:
    LITELLM_AVAILABLE = False
    print("Warning: litellm not available. LLM classifier will use fallback.")


class LLMEmotionClassifier:
    """LLM-based emotion classifier using MiniMax M2.7 or GLM-5.1"""
    
    def __init__(self, model: str = "minimax-m2.7", use_fallback: bool = True):
        """
        Initialize LLM classifier.
        
        Args:
            model: LLM model to use ('minimax-m2.7', 'zai/glm-5.1', 'anthropic/claude-sonnet-4-6')
            use_fallback: Whether to fall back to rule-based classifier on failure
        """
        self.model = model
        self.use_fallback = use_fallback
        self.litellm_base = os.getenv("LITELLM_BASE", "http://192.168.1.106:4000")
        self.api_key = os.getenv("LITELLM_KEY", "sk-labz0rz-master-key")
        
        # Model-specific configurations
        self.model_configs = {
            "minimax-m2.7": {
                "temperature": 0.1,
                "max_tokens": 500,
                "timeout": 30
            },
            "zai/glm-5.1": {
                "temperature": 0.1,
                "max_tokens": 500,
                "timeout": 45
            },
            "anthropic/claude-sonnet-4-6": {
                "temperature": 0.1,
                "max_tokens": 500,
                "timeout": 60
            }
        }
        
        # Default config
        self.config = self.model_configs.get(model, {
            "temperature": 0.1,
            "max_tokens": 500,
            "timeout": 30
        })
        
        # Initialize fallback classifier if needed
        if use_fallback:
            from emotion_classifier import RuleBasedEmotionClassifier
            self.fallback_classifier = RuleBasedEmotionClassifier()
        else:
            self.fallback_classifier = None
    
    def _create_prompt(self, text: str, context: Optional[Dict] = None) -> str:
        """Create classification prompt for LLM"""
        
        context_str = json.dumps(context or {}, indent=2)
        
        prompt = f"""Analyze the emotional content of this game memory text from Dark Pawns:

TEXT: "{text}"

CONTEXT: {context_str}

Classify the emotional valence using this schema:

1. CATEGORY (required): Choose exactly one:
   - "positive": Pleasant, rewarding, successful experiences
   - "negative": Unpleasant, punishing, failed experiences  
   - "neutral": Neither positive nor negative

2. INTENSITY (required): Scale 1-5 where:
   - 1: Subtle - barely noticeable emotional impact
   - 2: Mild - noticeable but not significant  
   - 3: Moderate - clearly felt emotional impact
   - 4: Strong - powerful emotional response
   - 5: Intense - overwhelming emotional experience

3. PRIMARY EMOTIONS (optional): List 0-3 emotions from: 
   joy, anger, fear, sadness, surprise, disgust, trust, anticipation

4. CONFIDENCE (required): 0.0-1.0, your confidence in this classification

5. EXPLANATION (optional): Brief explanation of your reasoning

Return ONLY valid JSON with these exact keys:
{{
  "category": "positive/negative/neutral",
  "intensity": 1-5,
  "primary_emotions": ["emotion1", "emotion2"],
  "confidence": 0.85,
  "explanation": "brief explanation"
}}

Do not include any other text, markdown, or formatting."""
        
        return prompt
    
    def _extract_json_from_response(self, response_text: str) -> Optional[Dict]:
        """Extract JSON from LLM response"""
        try:
            # Find JSON in response
            json_start = response_text.find('{')
            json_end = response_text.rfind('}') + 1
            
            if json_start >= 0 and json_end > json_start:
                json_str = response_text[json_start:json_end]
                
                # Clean up common issues
                json_str = re.sub(r',\s*}', '}', json_str)  # Trailing commas
                json_str = re.sub(r',\s*]', ']', json_str)  # Trailing commas in arrays
                
                result = json.loads(json_str)
                
                # Validate required fields
                required_fields = ['category', 'intensity', 'confidence']
                if not all(field in result for field in required_fields):
                    return None
                
                # Validate category
                if result['category'] not in ['positive', 'negative', 'neutral']:
                    return None
                
                # Validate intensity
                if not (1 <= result['intensity'] <= 5):
                    return None
                
                # Validate confidence
                if not (0.0 <= result['confidence'] <= 1.0):
                    return None
                
                # Ensure primary_emotions is a list
                if 'primary_emotions' not in result:
                    result['primary_emotions'] = []
                elif not isinstance(result['primary_emotions'], list):
                    result['primary_emotions'] = []
                
                # Validate primary emotions
                valid_emotions = ['joy', 'anger', 'fear', 'sadness', 'surprise', 
                                 'disgust', 'trust', 'anticipation']
                result['primary_emotions'] = [
                    e for e in result['primary_emotions'] 
                    if e in valid_emotions
                ][:3]  # Limit to 3
                
                return result
                
        except (json.JSONDecodeError, KeyError, TypeError) as e:
            print(f"JSON extraction error: {e}")
        
        return None
    
    def classify(self, text: str, context: Optional[Dict] = None) -> Dict:
        """Classify emotion using LLM"""
        
        if not LITELLM_AVAILABLE:
            print("Error: litellm not available. Using fallback.")
            return self._fallback_classify(text, context)
        
        prompt = self._create_prompt(text, context)
        
        try:
            response = completion(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                base_url=self.litellm_base,
                api_key=self.api_key,
                temperature=self.config["temperature"],
                max_tokens=self.config["max_tokens"],
                timeout=self.config["timeout"]
            )
            
            result_text = response.choices[0].message.content
            print(f"LLM raw response: {result_text[:200]}...")
            
            result = self._extract_json_from_response(result_text)
            
            if result:
                result["method"] = f"llm_{self.model}"
                return result
            else:
                print("Failed to extract valid JSON from LLM response")
                
        except Exception as e:
            print(f"LLM classification failed: {e}")
        
        # Fallback to rule-based classifier
        return self._fallback_classify(text, context)
    
    def _fallback_classify(self, text: str, context: Optional[Dict] = None) -> Dict:
        """Fallback to rule-based classification"""
        if self.fallback_classifier:
            result = self.fallback_classifier.classify(text, context)
            result["method"] = f"fallback_rule_based"
            return result
        else:
            # Minimal fallback
            return {
                "category": "neutral",
                "intensity": 1,
                "primary_emotions": [],
                "confidence": 0.3,
                "method": "minimal_fallback",
                "explanation": "LLM classifier failed and no fallback available"
            }
    
    def batch_classify(self, texts: List[str], contexts: Optional[List[Dict]] = None) -> List[Dict]:
        """Classify multiple texts"""
        results = []
        contexts = contexts or [{}] * len(texts)
        
        for i, (text, context) in enumerate(zip(texts, contexts)):
            print(f"Classifying {i+1}/{len(texts)}...")
            result = self.classify(text, context)
            results.append(result)
            
            # Small delay to avoid rate limiting
            import time
            time.sleep(0.5)
        
        return results


def test_classifier():
    """Test the LLM classifier with example texts"""
    
    if not LITELLM_AVAILABLE:
        print("litellm not available. Testing with mock data.")
        classifier = LLMEmotionClassifier(model="minimax-m2.7", use_fallback=True)
        
        # Mock the completion function for testing
        import unittest.mock as mock
        
        mock_response = mock.Mock()
        mock_response.choices = [mock.Mock()]
        mock_response.choices[0].message.content = '''{
  "category": "positive",
  "intensity": 4,
  "primary_emotions": ["joy", "trust"],
  "confidence": 0.92,
  "explanation": "Victory in combat evokes joy and trust in one's abilities"
}'''
        
        with mock.patch('emotion_llm_classifier.completion', return_value=mock_response):
            test_text = "Killed the dragon and took its treasure. Feeling invincible!"
            result = classifier.classify(test_text)
    else:
        classifier = LLMEmotionClassifier(model="minimax-m2.7", use_fallback=True)
        
        test_cases = [
            {
                "text": "Killed the dragon and took its treasure. Feeling invincible!",
                "context": {"event_type": "mob_kill", "valence": 3}
            },
            {
                "text": "Died to a rat in the sewers. Embarrassing.",
                "context": {"event_type": "mob_death", "valence": -3}
            },
            {
                "text": "Moved from room 100 to room 101.",
                "context": {"event_type": "room_visit", "valence": 0}
            }
        ]
        
        print("Testing LLMEmotionClassifier:")
        print("=" * 80)
        
        for test_case in test_cases:
            result = classifier.classify(test_case["text"], test_case["context"])
            print(f"Text: {test_case['text']}")
            print(f"Category: {result['category']} (intensity: {result['intensity']})")
            print(f"Primary emotions: {result['primary_emotions']}")
            print(f"Confidence: {result['confidence']}")
            print(f"Method: {result['method']}")
            if 'explanation' in result:
                print(f"Explanation: {result['explanation']}")
            print("-" * 80)


if __name__ == "__main__":
    test_classifier()
#!/usr/bin/env python3
"""
Rule-based emotion classifier for Dark Pawns narrative memory.

Simple keyword-based classifier that provides baseline emotional tagging.
"""

import re
from typing import Dict, List, Optional, Tuple
import json
from datetime import datetime


class RuleBasedEmotionClassifier:
    """Simple keyword-based emotion classifier"""
    
    def __init__(self):
        # Positive emotion keywords
        self.positive_keywords = [
            'victory', 'won', 'success', 'good', 'great', 'excellent',
            'happy', 'joy', 'pleased', 'satisfied', 'proud', 'triumph',
            'loot', 'treasure', 'reward', 'level up', 'stronger', 'better',
            'win', 'defeat', 'beat', 'crush', 'destroyed', 'dominated',
            'awesome', 'amazing', 'fantastic', 'wonderful', 'brilliant',
            'easy', 'simple', 'effortless', 'smooth', 'clean', 'perfect'
        ]
        
        # Negative emotion keywords
        self.negative_keywords = [
            'death', 'died', 'killed', 'lost', 'failed', 'bad', 'terrible',
            'angry', 'frustrated', 'scared', 'afraid', 'fear', 'sad',
            'disappointed', 'hurt', 'pain', 'damage', 'weak', 'poor',
            'awful', 'horrible', 'disaster', 'mistake', 'error', 'wrong',
            'difficult', 'hard', 'tough', 'struggle', 'suffer', 'bleeding',
            'low', 'danger', 'risk', 'almost died', 'barely', 'nearly',
            'escape', 'flee', 'run away', 'retreat', 'withdraw'
        ]
        
        # Primary emotion keywords
        self.emotion_keywords = {
            'joy': ['happy', 'joy', 'pleased', 'satisfied', 'proud', 'excited',
                   'delighted', 'ecstatic', 'thrilled', 'elated', 'jubilant'],
            'anger': ['angry', 'frustrated', 'irritated', 'rage', 'mad', 'furious',
                     'annoyed', 'outraged', 'resentful', 'bitter', 'hostile'],
            'fear': ['scared', 'afraid', 'fear', 'terrified', 'anxious', 'worried',
                    'nervous', 'panicked', 'horrified', 'dread', 'apprehensive'],
            'sadness': ['sad', 'disappointed', 'grief', 'loss', 'unhappy', 'depressed',
                       'melancholy', 'heartbroken', 'despair', 'hopeless', 'miserable'],
            'surprise': ['surprised', 'shocked', 'astonished', 'unexpected', 'amazed',
                        'stunned', 'astounded', 'startled', 'bewildered', 'confused'],
            'disgust': ['disgusted', 'revolted', 'gross', 'nasty', 'repulsed', 'sickened',
                       'contempt', 'distaste', 'revulsion', 'loathing', 'abhorrence'],
            'trust': ['trust', 'confident', 'reliable', 'safe', 'secure', 'certain',
                     'assured', 'convinced', 'faith', 'belief', 'dependable'],
            'anticipation': ['anticipate', 'expect', 'excited', 'looking forward',
                            'eager', 'hopeful', 'awaiting', 'prepared', 'ready']
        }
        
        # Negation words that invert meaning
        self.negation_words = ['not', 'never', 'no', "n't", 'without', 'lack']
        
        # Intensity modifiers
        self.intensity_modifiers = {
            'very': 2, 'extremely': 3, 'incredibly': 3, 'really': 2,
            'quite': 1, 'somewhat': 1, 'slightly': 0.5, 'barely': 0.5,
            'totally': 3, 'completely': 3, 'absolutely': 3
        }
    
    def _contains_negation(self, text: str, keyword: str) -> bool:
        """Check if keyword is negated in text"""
        pattern = rf'({"|".join(self.negation_words)})\s+\w*\s*{keyword}'
        return bool(re.search(pattern, text, re.IGNORECASE))
    
    def _get_intensity_modifier(self, text: str, keyword: str) -> float:
        """Check for intensity modifiers near keyword"""
        words = text.lower().split()
        keyword_index = -1
        
        # Find keyword position
        for i, word in enumerate(words):
            if keyword in word:
                keyword_index = i
                break
        
        if keyword_index == -1:
            return 1.0
        
        # Check words before keyword for modifiers
        for i in range(max(0, keyword_index - 3), keyword_index):
            if words[i] in self.intensity_modifiers:
                return self.intensity_modifiers[words[i]]
        
        return 1.0
    
    def classify(self, text: str, context: Optional[Dict] = None) -> Dict:
        """Classify emotion from text"""
        text_lower = text.lower()
        context = context or {}
        
        # Count keyword matches with negation check
        positive_count = 0
        negative_count = 0
        
        for kw in self.positive_keywords:
            if kw in text_lower and not self._contains_negation(text_lower, kw):
                positive_count += 1 * self._get_intensity_modifier(text_lower, kw)
        
        for kw in self.negative_keywords:
            if kw in text_lower and not self._contains_negation(text_lower, kw):
                negative_count += 1 * self._get_intensity_modifier(text_lower, kw)
        
        # Use context valence if available (from Postgres -3 to +3 scale)
        context_valence = context.get('valence', 0)
        if context_valence != 0:
            if context_valence > 0:
                positive_count += abs(context_valence)
            else:
                negative_count += abs(context_valence)
        
        # Determine category
        if positive_count > negative_count:
            category = "positive"
            # Base intensity on difference
            intensity_score = positive_count - negative_count
        elif negative_count > positive_count:
            category = "negative"
            intensity_score = negative_count - positive_count
        else:
            category = "neutral"
            intensity_score = 0
        
        # Map intensity score to 1-5 scale
        if intensity_score == 0:
            intensity = 1
        elif intensity_score < 2:
            intensity = 2
        elif intensity_score < 4:
            intensity = 3
        elif intensity_score < 6:
            intensity = 4
        else:
            intensity = 5
        
        # Identify primary emotions
        primary_emotions = []
        emotion_scores = {}
        
        for emotion, keywords in self.emotion_keywords.items():
            score = 0
            for kw in keywords:
                if kw in text_lower and not self._contains_negation(text_lower, kw):
                    score += 1 * self._get_intensity_modifier(text_lower, kw)
            
            if score > 0:
                emotion_scores[emotion] = score
        
        # Sort emotions by score and take top 3
        sorted_emotions = sorted(emotion_scores.items(), key=lambda x: x[1], reverse=True)
        primary_emotions = [emotion for emotion, score in sorted_emotions[:3]]
        
        # Calculate confidence
        total_keywords = positive_count + negative_count
        if total_keywords > 0:
            # Higher confidence with more keywords and clear category difference
            keyword_confidence = min(0.7, total_keywords / 10)
            difference_confidence = min(0.3, abs(positive_count - negative_count) / 10)
            confidence = keyword_confidence + difference_confidence
        else:
            # No keywords found
            confidence = 0.3
        
        # Adjust confidence based on context
        if context:
            confidence = min(0.9, confidence + 0.1)
        
        return {
            "category": category,
            "intensity": intensity,
            "primary_emotions": primary_emotions,
            "confidence": round(confidence, 2),
            "method": "rule_based",
            "scores": {
                "positive": round(positive_count, 2),
                "negative": round(negative_count, 2),
                "emotion_scores": emotion_scores
            }
        }


def test_classifier():
    """Test the classifier with example texts"""
    classifier = RuleBasedEmotionClassifier()
    
    test_cases = [
        "Killed the dragon and took its treasure. Feeling invincible!",
        "Died to a rat in the sewers. Embarrassing and frustrating.",
        "Moved from room 100 to room 101.",
        "Barely survived the orc attack. Almost died but managed to flee.",
        "Found an amazing magical sword! This is fantastic!",
        "The goblin ambush was terrifying. I was so scared.",
        "Not happy with the loot from that chest. Disappointed.",
        "Absolutely destroyed the troll king. Total victory!",
    ]
    
    print("Testing RuleBasedEmotionClassifier:")
    print("=" * 80)
    
    for text in test_cases:
        result = classifier.classify(text)
        print(f"Text: {text}")
        print(f"Category: {result['category']} (intensity: {result['intensity']})")
        print(f"Primary emotions: {result['primary_emotions']}")
        print(f"Confidence: {result['confidence']}")
        print(f"Scores: {result['scores']}")
        print("-" * 80)


if __name__ == "__main__":
    test_classifier()
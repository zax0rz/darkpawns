#!/usr/bin/env python3
"""
Integration tests for Dark Pawns AI system.
"""

import pytest
import asyncio
import json
import os
import sys
from unittest.mock import Mock, patch, AsyncMock

# Add parent directory to path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../../'))

try:
    from pkg.ai.brain import AIContext, AIBrain
    from pkg.ai.behaviors import BehaviorManager, Behavior
    HAS_GO_AI = True
except ImportError:
    HAS_GO_AI = False
    print("Warning: Go AI modules not available for import")

# Mock AI services for testing
class MockAIService:
    """Mock AI service for testing."""
    
    def __init__(self):
        self.responses = []
        self.calls = []
    
    async def generate_response(self, prompt, context=None):
        """Generate a mock AI response."""
        self.calls.append({
            'prompt': prompt,
            'context': context
        })
        
        if self.responses:
            return self.responses.pop(0)
        
        # Default mock response
        return {
            'text': 'I am a mock AI response.',
            'confidence': 0.8,
            'reasoning': 'Mock reasoning for test purposes.'
        }
    
    async def analyze_sentiment(self, text):
        """Analyze sentiment of text."""
        return {
            'sentiment': 'neutral',
            'score': 0.0,
            'emotions': {}
        }
    
    async def extract_entities(self, text):
        """Extract entities from text."""
        return {
            'entities': [],
            'relationships': []
        }

@pytest.mark.skipif(not HAS_GO_AI, reason="Go AI modules not available")
class TestAIIntegration:
    """Test AI system integration."""
    
    @pytest.fixture
    def mock_ai_service(self):
        """Create a mock AI service."""
        return MockAIService()
    
    @pytest.fixture
    def ai_context(self):
        """Create an AI context for testing."""
        return AIContext(
            player_id="test-player-123",
            player_name="TestPlayer",
            room_id="room-1",
            room_name="Test Room",
            game_state={
                'health': 100,
                'mana': 50,
                'inventory': ['sword', 'potion'],
                'location': 'Test Room'
            },
            memory_context="Test memory context",
            recent_events=["Entered room", "Saw a chest"]
        )
    
    @pytest.mark.asyncio
    async def test_ai_response_generation(self, mock_ai_service, ai_context):
        """Test AI response generation."""
        
        # Configure mock response
        mock_ai_service.responses.append({
            'text': 'I see a chest in the room. Should we open it?',
            'confidence': 0.9,
            'reasoning': 'Player mentioned chest, suggesting interaction.'
        })
        
        # Generate response
        response = await mock_ai_service.generate_response(
            prompt="What should I do next?",
            context=ai_context.to_dict()
        )
        
        assert response['text'] == 'I see a chest in the room. Should we open it?'
        assert response['confidence'] == 0.9
        assert 'chest' in response['reasoning'].lower()
        assert len(mock_ai_service.calls) == 1
    
    @pytest.mark.asyncio
    async def test_ai_sentiment_analysis(self, mock_ai_service):
        """Test AI sentiment analysis."""
        
        sentiment = await mock_ai_service.analyze_sentiment(
            "I am very happy to be here!"
        )
        
        assert 'sentiment' in sentiment
        assert 'score' in sentiment
        assert 'emotions' in sentiment
    
    @pytest.mark.asyncio
    async def test_ai_entity_extraction(self, mock_ai_service):
        """Test AI entity extraction."""
        
        entities = await mock_ai_service.extract_entities(
            "The dragon guards the treasure in the cave."
        )
        
        assert 'entities' in entities
        assert 'relationships' in entities
        assert isinstance(entities['entities'], list)
        assert isinstance(entities['relationships'], list)
    
    def test_ai_context_serialization(self, ai_context):
        """Test AI context serialization."""
        
        context_dict = ai_context.to_dict()
        
        assert context_dict['player_id'] == "test-player-123"
        assert context_dict['player_name'] == "TestPlayer"
        assert context_dict['room_id'] == "room-1"
        assert context_dict['room_name'] == "Test Room"
        assert context_dict['game_state']['health'] == 100
        assert context_dict['memory_context'] == "Test memory context"
        assert len(context_dict['recent_events']) == 2
    
    def test_ai_context_from_dict(self):
        """Test creating AI context from dictionary."""
        
        context_dict = {
            'player_id': "player-456",
            'player_name': "AnotherPlayer",
            'room_id': "room-2",
            'room_name': "Another Room",
            'game_state': {
                'health': 75,
                'mana': 25,
                'inventory': ['shield', 'key'],
                'location': 'Another Room'
            },
            'memory_context': "Different memory",
            'recent_events': ["Killed goblin", "Found key"]
        }
        
        context = AIContext.from_dict(context_dict)
        
        assert context.player_id == "player-456"
        assert context.player_name == "AnotherPlayer"
        assert context.room_name == "Another Room"
        assert context.game_state['health'] == 75
        assert context.memory_context == "Different memory"
        assert len(context.recent_events) == 2

@pytest.mark.skipif(not HAS_GO_AI, reason="Go AI modules not available")
class TestBehaviorSystem:
    """Test behavior system integration."""
    
    @pytest.fixture
    def behavior_manager(self):
        """Create a behavior manager."""
        return BehaviorManager()
    
    @pytest.fixture
    def sample_behaviors(self):
        """Create sample behaviors for testing."""
        return [
            Behavior(
                name="explore",
                description="Explore the environment",
                priority=1,
                conditions=["not_in_combat", "health_above_50"],
                actions=["move_random", "examine_room"]
            ),
            Behavior(
                name="fight",
                description="Engage in combat",
                priority=10,
                conditions=["in_combat", "health_above_20"],
                actions=["attack_nearest", "use_best_skill"]
            ),
            Behavior(
                name="flee",
                description="Flee from danger",
                priority=5,
                conditions=["health_below_30", "in_combat"],
                actions=["move_to_safe", "use_escape_skill"]
            ),
            Behavior(
                name="rest",
                description="Rest and recover",
                priority=2,
                conditions=["health_below_50", "not_in_combat", "safe_location"],
                actions=["sit", "meditate", "use_potion"]
            )
        ]
    
    def test_behavior_creation(self):
        """Test behavior creation."""
        
        behavior = Behavior(
            name="test_behavior",
            description="Test behavior",
            priority=3,
            conditions=["test_condition"],
            actions=["test_action"]
        )
        
        assert behavior.name == "test_behavior"
        assert behavior.description == "Test behavior"
        assert behavior.priority == 3
        assert behavior.conditions == ["test_condition"]
        assert behavior.actions == ["test_action"]
    
    def test_behavior_manager_initialization(self, behavior_manager):
        """Test behavior manager initialization."""
        
        assert behavior_manager.behaviors == {}
        assert behavior_manager.active_behavior is None
    
    def test_behavior_registration(self, behavior_manager, sample_behaviors):
        """Test behavior registration."""
        
        for behavior in sample_behaviors:
            behavior_manager.register_behavior(behavior)
        
        assert len(behavior_manager.behaviors) == 4
        assert "explore" in behavior_manager.behaviors
        assert "fight" in behavior_manager.behaviors
        assert "flee" in behavior_manager.behaviors
        assert "rest" in behavior_manager.behaviors
    
    def test_behavior_evaluation(self, behavior_manager, sample_behaviors):
        """Test behavior evaluation."""
        
        # Register behaviors
        for behavior in sample_behaviors:
            behavior_manager.register_behavior(behavior)
        
        # Test with different game states
        test_cases = [
            {
                'name': 'Healthy and safe',
                'game_state': {
                    'in_combat': False,
                    'health': 80,
                    'safe_location': True
                },
                'expected_behavior': 'explore'  # Highest priority that matches
            },
            {
                'name': 'In combat and healthy',
                'game_state': {
                    'in_combat': True,
                    'health': 80,
                    'safe_location': False
                },
                'expected_behavior': 'fight'  # Highest priority in combat
            },
            {
                'name': 'Low health in combat',
                'game_state': {
                    'in_combat': True,
                    'health': 25,
                    'safe_location': False
                },
                'expected_behavior': 'flee'  # flee has priority 5, fight has 10 but health too low
            },
            {
                'name': 'Low health but safe',
                'game_state': {
                    'in_combat': False,
                    'health': 40,
                    'safe_location': True
                },
                'expected_behavior': 'rest'  # Only rest matches all conditions
            }
        ]
        
        for test_case in test_cases:
            # Mock condition checking
            def mock_check_condition(condition, game_state):
                return game_state.get(condition, False)
            
            # Evaluate behavior
            selected = behavior_manager.evaluate_behavior(
                test_case['game_state'],
                condition_checker=mock_check_condition
            )
            
            if test_case['expected_behavior']:
                assert selected.name == test_case['expected_behavior']
            else:
                assert selected is None
    
    def test_behavior_execution(self, behavior_manager):
        """Test behavior execution."""
        
        # Create a test behavior
        test_behavior = Behavior(
            name="test_execute",
            description="Test execution",
            priority=1,
            conditions=[],
            actions=["action1", "action2", "action3"]
        )
        
        behavior_manager.register_behavior(test_behavior)
        
        # Mock action executor
        executed_actions = []
        
        def mock_execute_action(action, game_state):
            executed_actions.append(action)
            return f"Executed {action}"
        
        # Execute behavior
        results = behavior_manager.execute_behavior(
            "test_execute",
            game_state={},
            action_executor=mock_execute_action
        )
        
        assert len(results) == 3
        assert "Executed action1" in results
        assert "Executed action2" in results
        assert "Executed action3" in results
        assert executed_actions == ["action1", "action2", "action3"]
    
    def test_behavior_priority_ordering(self, behavior_manager):
        """Test that behaviors are evaluated in priority order."""
        
        # Create behaviors with different priorities
        behaviors = [
            Behavior(name="low", priority=1, conditions=["condition"], actions=[]),
            Behavior(name="medium", priority=5, conditions=["condition"], actions=[]),
            Behavior(name="high", priority=10, conditions=["condition"], actions=[])
        ]
        
        for behavior in behaviors:
            behavior_manager.register_behavior(behavior)
        
        # Mock condition checker that always returns True
        def mock_check_condition(condition, game_state):
            return True
        
        # Evaluate - should return highest priority behavior that matches
        selected = behavior_manager.evaluate_behavior(
            game_state={},
            condition_checker=mock_check_condition
        )
        
        assert selected.name == "high"  # Highest priority
    
    def test_behavior_condition_failure(self, behavior_manager):
        """Test behavior evaluation when conditions don't match."""
        
        behavior = Behavior(
            name="conditional",
            description="Requires specific condition",
            priority=1,
            conditions=["special_condition"],
            actions=[]
        )
        
        behavior_manager.register_behavior(behavior)
        
        # Mock condition checker that returns False
        def mock_check_condition(condition, game_state):
            return False
        
        # Evaluate - should return None since condition fails
        selected = behavior_manager.evaluate_behavior(
            game_state={},
            condition_checker=mock_check_condition
        )
        
        assert selected is None

@pytest.mark.skipif(not HAS_GO_AI, reason="Go AI modules not available")
class TestAIMemoryIntegration:
    """Test AI memory system integration."""
    
    @pytest.fixture
    def mock_memory_system(self):
        """Create a mock memory system."""
        memory = {
            'events': [],
            'entities': {},
            'relationships': {},
            'goals': []
        }
        
        class MockMemory:
            def __init__(self):
                self.memory = memory.copy()
                self.calls = []
            
            async def store_event(self, event_type, description, importance=1):
                """Store an event in memory."""
                self.calls.append(('store_event', event_type, description, importance))
                self.memory['events'].append({
                    'type': event_type,
                    'description': description,
                    'importance': importance,
                    'timestamp': '2024-01-01T00:00:00Z'
                })
                return True
            
            async def recall_events(self, event_type=None, limit=10):
                """Recall events from memory."""
                self.calls.append(('recall_events', event_type, limit))
                
                events = self.memory['events']
                if event_type:
                    events = [e for e in events if e['type'] == event_type]
                
                return events[:limit]
            
            async def get_context(self, current_situation):
                """Get relevant memory context."""
                self.calls.append(('get_context', current_situation))
                
                # Return a summary of relevant events
                relevant = []
                for event in self.memory['events'][-5:]:  # Last 5 events
                    if any(keyword in event['description'].lower() 
                           for keyword in current_situation.lower().split()):
                        relevant.append(event)
                
                return {
                    'summary': f"Relevant memories: {len(relevant)}",
                    'events': relevant
                }
        
        return MockMemory()
    
    @pytest.mark.asyncio
    async def test_memory_event_storage(self, mock_memory_system):
        """Test storing events in memory."""
        
        # Store some events
        await mock_memory_system.store_event(
            event_type="combat",
            description="Defeated a goblin",
            importance=3
        )
        
        await mock_memory_system.store_event(
            event_type="exploration",
            description="Found a hidden passage",
            importance=2
        )
        
        await mock_memory_system.store_event(
            event_type="dialogue",
            description="Talked to the blacksmith",
            importance=1
        )
        
        # Verify storage
        assert len(mock_memory_system.memory['events']) == 3
        assert mock_memory_system.memory['events'][0]['type'] == "combat"
        assert "goblin" in mock_memory_system.memory['events'][0]['description']
        assert mock_memory_system.memory['events'][0]['importance'] == 3
        
        assert len(mock_memory_system.calls) == 3
    
    @pytest.mark.asyncio
    async def test_memory_event_recall(self, mock_memory_system):
        """Test recalling events from memory."""
        
        # First store some events
        events = [
            ("combat", "Killed dragon", 5),
            ("exploration", "Entered cave", 2),
            ("combat", "Fought skeletons", 3),
            ("dialogue", "Met wizard", 2),
            ("combat", "Defeated troll", 4)
        ]
        
        for event_type, description, importance in events:
            await mock_memory_system.store_event(event_type, description, importance)
        
        # Recall all events
        all_events = await mock_memory_system.recall_events()
        assert len(all_events) == 5
        
        # Recall only combat events
        combat_events = await mock_memory_system.recall_events(event_type="combat")
        assert len(combat_events) == 3
        assert all(e['type'] == "combat" for e in combat_events)
        
        # Recall with limit
        limited_events = await mock_memory_system.recall_events(limit=2)
        assert len(limited_events) == 2
    
    @pytest.mark.asyncio
    async def test_memory_context_retrieval(self, mock_memory_system):
        """Test retrieving memory context."""
        
        # Store events
        await mock_memory_system.store_event("combat", "Fought goblins in forest", 3)
        await mock_memory_system.store_event("exploration", "Found ancient ruins", 2)
        await mock_memory_system.store_event("combat", "Defeated forest guardian", 4)
        await mock_memory_system.store_event("dialogue", "Met elf in forest", 2)
        
        # Get context for forest situation
        context = await mock_memory_system.get_context("entering dark forest")
        
        assert 'summary' in context
        assert 'events' in context
        assert isinstance(context['events'], list)
        
        # Should find events mentioning forest
        forest_events = [e for e in context['events'] if 'forest' in e['description'].lower()]
        assert len(forest_events) > 0

@pytest.mark.skipif(not HAS_GO_AI, reason="Go AI modules not available")
class TestAIDecisionMaking:
    """Test AI decision making integration."""
    
    @pytest.fixture
    def decision_maker(self):
        """Create a decision maker for testing."""
        
        class MockDecisionMaker:
            def __init__(self):
                self.decisions = []
                self.contexts = []
            
            async def make_decision(self, context, options):
                """Make a decision based on context and options."""
                self.contexts.append(context)
                self.decisions.append(options)
                
                # Simple decision logic for testing
                if 'combat' in context.get('situation', '').lower():
                    # In combat, choose attack options
                    for option in options:
                        if 'attack' in option.get('action', '').lower():
                            return option
                
                # Default: choose first option
                return options[0] if options else None
        
        return MockDecisionMaker()
    
    @pytest.mark.asyncio
    async def test_combat_decision(self, decision_maker):
        """Test decision making in combat."""
        
        context = {
            'situation': 'In combat with goblin',
            'health': 75,
            'enemy_health': 50,
            'options_available': ['attack', 'defend', 'flee']
        }
        
        options = [
            {'action': 'attack', 'target': 'goblin', 'risk': 'medium'},
            {'action': 'defend', 'effect': 'reduce damage', 'risk': 'low'},
            {'action': 'flee', 'chance': 0.7, 'risk': 'low'}
        ]
        
        decision = await decision_maker.make_decision(context, options)
        
        assert decision['action'] == 'attack'
        assert decision['target'] == 'goblin'
        assert len(decision_maker.contexts) == 1
        assert len(decision_maker.decisions) == 1
    
    @pytest.mark.asyncio
    async def test_exploration_decision(self, decision_maker):
        """Test decision making during exploration."""
        
        context = {
            'situation': 'Exploring dungeon',
            'paths': ['north', 'east', 'west'],
            'resources': ['torch', 'rope']
        }
        
        options = [
            {'action': 'go_north', 'description': 'Dark corridor'},
            {'action': 'go_east', 'description': 'Light from ahead'},
            {'action': 'go_west', 'description': 'Strange noises'}
        ]
        
        decision = await decision_maker.make_decision(context, options)
        
        # Should choose first option by default
        assert decision['action'] == 'go_north'
        assert 'Dark corridor' in decision['description']

if __name__ == "__main__":
    # Run tests
    pytest.main([__file__, "-v"])
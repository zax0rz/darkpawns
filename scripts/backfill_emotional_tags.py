#!/usr/bin/env python3
"""
Backfill emotional tags for existing Dark Pawns narrative memories.

This script reads existing memories from Postgres, tags them with emotional valence,
and updates the database with the results.
"""

import os
import sys
import json
import argparse
from typing import List, Dict, Optional
from datetime import datetime

# Add parent directory to path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

try:
    import psycopg2
    from psycopg2.extras import RealDictCursor
except ImportError:
    print("Error: psycopg2 not installed. Install with: pip install psycopg2-binary")
    sys.exit(1)

try:
    from scripts.emotion_tagger import EmotionTagger, update_postgres_memory
except ImportError:
    # Try relative import
    try:
        from emotion_tagger import EmotionTagger, update_postgres_memory
    except ImportError:
        print("Error: Could not import EmotionTagger. Make sure emotion_tagger.py is in the same directory.")
        sys.exit(1)


class EmotionalTagBackfiller:
    """Backfill emotional tags for existing memories"""
    
    def __init__(self, db_url: str, use_llm: bool = False, batch_size: int = 50):
        """
        Initialize backfiller.
        
        Args:
            db_url: PostgreSQL connection URL
            use_llm: Whether to use LLM classifier (slower but more accurate)
            batch_size: Number of memories to process in each batch
        """
        self.db_url = db_url
        self.batch_size = batch_size
        self.tagger = EmotionTagger(use_llm=use_llm, llm_threshold=0.6)
        
        # Statistics
        self.stats = {
            "total_memories": 0,
            "processed_memories": 0,
            "successful_updates": 0,
            "failed_updates": 0,
            "skipped_memories": 0,
            "start_time": None,
            "end_time": None,
            "category_distribution": {"positive": 0, "negative": 0, "neutral": 0}
        }
    
    def connect_db(self):
        """Connect to PostgreSQL database"""
        try:
            conn = psycopg2.connect(self.db_url)
            conn.autocommit = False
            return conn
        except Exception as e:
            print(f"Error connecting to database: {e}")
            raise
    
    def get_untagged_memories(self, conn, limit: Optional[int] = None) -> List[Dict]:
        """Get memories that don't have emotional tags yet"""
        cursor = conn.cursor(cursor_factory=RealDictCursor)
        
        query = """
        SELECT id, summary, valence, room_name, related_entity, event_type,
               agent_name, created_at, salience
        FROM agent_narrative_memory
        WHERE emotion_category IS NULL
        ORDER BY created_at DESC
        """
        
        if limit:
            query += f" LIMIT {limit}"
        
        cursor.execute(query)
        memories = cursor.fetchall()
        cursor.close()
        
        return memories
    
    def create_context(self, memory: Dict) -> Dict:
        """Create context dictionary for emotion classification"""
        return {
            "valence": memory.get("valence", 0),
            "room_name": memory.get("room_name", ""),
            "related_entity": memory.get("related_entity", ""),
            "event_type": memory.get("event_type", ""),
            "agent_name": memory.get("agent_name", ""),
            "salience": memory.get("salience", 1.0),
            "created_at": memory.get("created_at")
        }
    
    def process_memory(self, conn, memory: Dict) -> bool:
        """Process and tag a single memory"""
        try:
            # Create context
            context = self.create_context(memory)
            text = memory["summary"]
            
            # Tag memory
            tags = self.tagger.tag_memory(text, context)
            
            # Update statistics
            category = tags["category"]
            if category in self.stats["category_distribution"]:
                self.stats["category_distribution"][category] += 1
            
            # Update database
            cursor = conn.cursor()
            success = update_postgres_memory(cursor, memory["id"], tags)
            
            if success:
                self.stats["successful_updates"] += 1
                conn.commit()
                
                # Print progress
                print(f"  ✓ Memory {memory['id']}: {tags['category']} ({tags['intensity']}) "
                      f"[{tags.get('method', 'unknown')}]")
                return True
            else:
                self.stats["failed_updates"] += 1
                conn.rollback()
                print(f"  ✗ Memory {memory['id']}: Update failed")
                return False
                
        except Exception as e:
            self.stats["failed_updates"] += 1
            conn.rollback()
            print(f"  ✗ Memory {memory['id']}: Error - {e}")
            return False
    
    def process_batch(self, conn, memories: List[Dict]) -> Dict:
        """Process a batch of memories"""
        batch_stats = {
            "total": len(memories),
            "successful": 0,
            "failed": 0,
            "categories": {"positive": 0, "negative": 0, "neutral": 0}
        }
        
        print(f"Processing batch of {len(memories)} memories...")
        
        for i, memory in enumerate(memories, 1):
            print(f"[{i}/{len(memories)}] ", end="")
            
            success = self.process_memory(conn, memory)
            
            if success:
                batch_stats["successful"] += 1
            else:
                batch_stats["failed"] += 1
            
            self.stats["processed_memories"] += 1
        
        return batch_stats
    
    def run(self, limit: Optional[int] = None, dry_run: bool = False):
        """Run the backfill process"""
        print("Starting emotional tag backfill...")
        print(f"Database: {self.db_url}")
        print(f"Use LLM: {self.tagger.use_llm}")
        print(f"Batch size: {self.batch_size}")
        print(f"Dry run: {dry_run}")
        print("=" * 80)
        
        self.stats["start_time"] = datetime.now()
        
        try:
            conn = self.connect_db()
            
            # Get untagged memories
            memories = self.get_untagged_memories(conn, limit)
            self.stats["total_memories"] = len(memories)
            
            if not memories:
                print("No untagged memories found.")
                return
            
            print(f"Found {len(memories)} untagged memories.")
            
            if dry_run:
                print("Dry run mode - would process:")
                for memory in memories[:5]:  # Show first 5 as sample
                    context = self.create_context(memory)
                    tags = self.tagger.tag_memory(memory["summary"], context)
                    print(f"  Memory {memory['id']}: {tags['category']} ({tags['intensity']})")
                print("... and more")
                return
            
            # Process in batches
            for i in range(0, len(memories), self.batch_size):
                batch = memories[i:i + self.batch_size]
                batch_num = i // self.batch_size + 1
                total_batches = (len(memories) + self.batch_size - 1) // self.batch_size
                
                print(f"\nBatch {batch_num}/{total_batches}")
                print("-" * 40)
                
                batch_stats = self.process_batch(conn, batch)
                
                # Print batch summary
                print(f"\nBatch {batch_num} summary:")
                print(f"  Successful: {batch_stats['successful']}/{batch_stats['total']}")
                print(f"  Failed: {batch_stats['failed']}/{batch_stats['total']}")
                
                # Print tagger statistics
                tagger_stats = self.tagger.get_statistics()
                print(f"  Tagger stats: {tagger_stats['rule_based_classifications']} rule-based, "
                      f"{tagger_stats['llm_classifications']} LLM")
                
                # Small delay between batches
                import time
                if i + self.batch_size < len(memories):
                    print("Pausing for 2 seconds...")
                    time.sleep(2)
            
            conn.close()
            
        except Exception as e:
            print(f"Error during backfill: {e}")
            import traceback
            traceback.print_exc()
        
        finally:
            self.stats["end_time"] = datetime.now()
            self.print_summary()
    
    def print_summary(self):
        """Print summary of backfill process"""
        print("\n" + "=" * 80)
        print("BACKFILL SUMMARY")
        print("=" * 80)
        
        duration = self.stats["end_time"] - self.stats["start_time"]
        
        print(f"Total memories: {self.stats['total_memories']}")
        print(f"Processed: {self.stats['processed_memories']}")
        print(f"Successful updates: {self.stats['successful_updates']}")
        print(f"Failed updates: {self.stats['failed_updates']}")
        print(f"Skipped: {self.stats['skipped_memories']}")
        print(f"Duration: {duration}")
        
        print("\nCategory distribution:")
        for category, count in self.stats["category_distribution"].items():
            if self.stats["processed_memories"] > 0:
                percentage = count / self.stats["processed_memories"] * 100
                print(f"  {category}: {count} ({percentage:.1f}%)")
            else:
                print(f"  {category}: {count}")
        
        # Print tagger statistics
        print("\nTagger statistics:")
        tagger_stats = self.tagger.get_statistics()
        for key, value in tagger_stats.items():
            if isinstance(value, dict):
                print(f"  {key}:")
                for subkey, subvalue in value.items():
                    print(f"    {subkey}: {subvalue}")
            else:
                print(f"  {key}: {value}")
        
        print("\n" + "=" * 80)


def main():
    """Main function"""
    parser = argparse.ArgumentParser(description="Backfill emotional tags for Dark Pawns memories")
    parser.add_argument("--db-url", required=True, help="PostgreSQL connection URL")
    parser.add_argument("--limit", type=int, help="Limit number of memories to process")
    parser.add_argument("--batch-size", type=int, default=50, help="Batch size for processing")
    parser.add_argument("--use-llm", action="store_true", help="Use LLM classifier (slower but more accurate)")
    parser.add_argument("--dry-run", action="store_true", help="Dry run without updating database")
    
    args = parser.parse_args()
    
    # Create backfiller
    backfiller = EmotionalTagBackfiller(
        db_url=args.db_url,
        use_llm=args.use_llm,
        batch_size=args.batch_size
    )
    
    # Run backfill
    backfiller.run(limit=args.limit, dry_run=args.dry_run)


if __name__ == "__main__":
    main()
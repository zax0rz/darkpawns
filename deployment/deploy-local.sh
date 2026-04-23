#!/bin/bash

# Dark Pawns Local Deployment Script
set -e

echo "🚀 Deploying Dark Pawns locally..."

# Check if Docker and Docker Compose are installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from template..."
    cat > .env << EOF
# Dark Pawns Environment Variables
WORLD_DIR=./lib
AI_API_KEY=REPLACE_WITH_SECURE_RANDOM_KEY
MEM0_API_KEY=
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# Optional: Override database credentials
# POSTGRES_PASSWORD=your_password_here
EOF
    echo "✅ Created .env file. Please edit it to add your API keys."
fi

# Build and start services
echo "🔨 Building Docker images..."
docker-compose build

echo "🚀 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are running
echo "🔍 Checking service status..."
docker-compose ps

echo "✅ Dark Pawns is now running!"
echo ""
echo "📊 Services:"
echo "  - Game Server: http://localhost:8080"
echo "  - PostgreSQL: localhost:5432 (database: darkpawns)"
echo "  - Redis: localhost:6379"
echo ""
echo "🔌 Connect via WebSocket: ws://localhost:8080/ws"
echo ""
echo "📝 To view logs:"
echo "  docker-compose logs -f server"
echo "  docker-compose logs -f ai-agent"
echo ""
echo "🛑 To stop: docker-compose down"
echo "🔄 To rebuild: docker-compose up -d --build"
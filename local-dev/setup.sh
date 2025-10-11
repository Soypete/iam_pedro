#!/bin/bash

set -e

echo "=== Pedro Local Development Setup ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v ollama &> /dev/null; then
    echo "❌ Ollama is not installed!"
    echo ""
    echo "Please install Ollama first:"
    echo "  macOS:   brew install ollama"
    echo "  Linux:   curl -fsSL https://ollama.com/install.sh | sh"
    echo "  Manual:  https://ollama.com/download"
    exit 1
fi
echo "✅ Ollama is installed"

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed!"
    echo "Please install Docker Desktop: https://www.docker.com/products/docker-desktop"
    exit 1
fi
echo "✅ Docker is installed"

if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed or too old!"
    echo "Please install Docker Compose v2+: https://docs.docker.com/compose/install/"
    exit 1
fi
echo "✅ Docker Compose is installed"

echo ""
echo "Checking if Ollama is running..."
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "⚠️  Ollama is not running!"
    echo ""
    echo "Starting Ollama in the background..."
    echo "Note: On macOS, Ollama may start automatically. If not, run 'ollama serve' in another terminal."

    # Try to start Ollama (works on Linux, may not work on macOS)
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        nohup ollama serve > /tmp/ollama.log 2>&1 &
        sleep 2
    else
        echo ""
        echo "Please start Ollama manually in another terminal:"
        echo "  ollama serve"
        echo ""
        read -p "Press Enter when Ollama is running..."
    fi
fi

# Verify Ollama is accessible
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✅ Ollama is running"
else
    echo "❌ Cannot connect to Ollama at http://localhost:11434"
    echo "Please make sure 'ollama serve' is running in another terminal"
    exit 1
fi

echo ""
echo "Pulling Qwen model for local development..."
echo "This may take a few minutes depending on your internet connection..."
echo ""

# Pull the recommended model
ollama pull qwen2.5-coder:7b-instruct

echo ""
echo "✅ Model pulled successfully!"
echo ""

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp .env.template .env
    echo "⚠️  Please edit .env and add your configuration values"
fi

# Create prod.env if it doesn't exist
if [ ! -f ../prod.env ]; then
    echo "Creating ../prod.env file from template..."
    cp prod.env.template ../prod.env
    echo "⚠️  Please edit ../prod.env with your secrets"
    echo ""
    echo "Options:"
    echo "  1. Use 1Password references: op://vault/item/field"
    echo "  2. Use raw values: your_actual_token_here"
fi

echo ""
echo "=== Setup Complete! ==="
echo ""
echo "Next steps:"
echo "1. Edit .env and ../prod.env with your configuration"
echo "2. Make sure 'ollama serve' is running in another terminal"
echo "3. Run: ./run.sh"
echo ""
echo "Available models in Ollama:"
ollama list
echo ""
echo "To pull different models:"
echo "  ollama pull qwen2.5-coder:3b-instruct  # Smaller, faster (4GB RAM)"
echo "  ollama pull qwen2.5-coder:14b-instruct # Larger, better (16GB RAM)"

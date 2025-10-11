#!/bin/bash

set -e

echo "=== Stopping Pedro Local Development Environment ==="
echo ""

# Stop all services
docker compose --profile twitch down

echo ""
echo "✅ All services stopped"
echo ""
echo "To remove volumes (database data):"
echo "  docker compose down -v"
echo ""
echo "To remove images:"
echo "  docker compose down --rmi all"
echo ""
echo "Note: Ollama is still running. To stop Ollama:"
echo "  macOS: Quit Ollama from menu bar"
echo "  Linux: pkill ollama"

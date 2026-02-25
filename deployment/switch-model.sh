#!/bin/bash

# Switch the active llama.cpp model on pedrogpt.
# Uses llama-server's built-in --hf-repo/--hf-file flags so the server
# downloads and caches the model itself (no manual wget needed).
# Models are cached in HF_HOME (/opt/models/cache) on the 2TB drive.
#
# pedrogpt hardware: 32GB VRAM + 64GB RAM
#
# pedro models (verified):
#   unsloth/gpt-oss-20b-GGUF                       gpt-oss-20b-Q4_K_M.gguf          11.6 GB
#   unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF      Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf  18.6 GB MoE
#   unsloth/Qwen3.5-35B-A3B-GGUF                   Qwen3.5-35B-A3B-Q4_K_M.gguf      21.2 GB MoE
#
# Usage:
#   ./switch-model.sh <hf-repo> <hf-file>
#   ./switch-model.sh list
#
# Examples:
#   ./switch-model.sh unsloth/gpt-oss-20b-GGUF gpt-oss-20b-Q4_K_M.gguf
#   ./switch-model.sh unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf
#   ./switch-model.sh unsloth/Qwen3.5-35B-A3B-GGUF Qwen3.5-35B-A3B-Q4_K_M.gguf

set -euo pipefail

ENV_FILE="/etc/llama-server.env"
SUBCOMMAND="${1:-}"

# ---------------------------------------------------------------------------
# list: show available pedro models
# ---------------------------------------------------------------------------
if [[ "$SUBCOMMAND" == "list" ]]; then
  echo "pedro models:"
  echo ""
  echo "  unsloth/gpt-oss-20b-GGUF"
  echo "    gpt-oss-20b-Q4_K_M.gguf          11.6 GB"
  echo "    gpt-oss-20b-Q8_0.gguf            12.1 GB  (near-lossless, also fits)"
  echo ""
  echo "  unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF"
  echo "    Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf  18.6 GB  MoE"
  echo "    Qwen3-Coder-30B-A3B-Instruct-Q5_K_M.gguf  21.7 GB  MoE"
  echo ""
  echo "  unsloth/Qwen3.5-35B-A3B-GGUF"
  echo "    Qwen3.5-35B-A3B-Q4_K_M.gguf      21.2 GB  MoE"
  echo "    Qwen3.5-35B-A3B-Q5_K_M.gguf      24.8 GB  MoE"
  echo ""
  echo "Cached models in /opt/models/cache:"
  find /opt/models/cache -name "*.gguf" -exec ls -lh {} \; 2>/dev/null || echo "  (none yet)"
  exit 0
fi

# ---------------------------------------------------------------------------
# switch: update env and restart service
# ---------------------------------------------------------------------------
HF_REPO="${1:-}"
HF_FILE="${2:-}"

if [[ -z "$HF_REPO" || -z "$HF_FILE" ]]; then
  echo "Usage: $0 <hf-repo> <hf-file>"
  echo "       $0 list"
  echo ""
  echo "Examples:"
  echo "  $0 unsloth/gpt-oss-20b-GGUF                      gpt-oss-20b-Q4_K_M.gguf"
  echo "  $0 unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF     Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"
  echo "  $0 unsloth/Qwen3.5-35B-A3B-GGUF                  Qwen3.5-35B-A3B-Q4_K_M.gguf"
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: $ENV_FILE not found. Run setup-llama-cpp-ubuntu.sh first."
  exit 1
fi

echo "=== Switching model ==="
echo "Repo: $HF_REPO"
echo "File: $HF_FILE"
echo ""

sudo sed -i "s|^HF_REPO=.*|HF_REPO=$HF_REPO|" "$ENV_FILE"
sudo sed -i "s|^HF_FILE=.*|HF_FILE=$HF_FILE|" "$ENV_FILE"

echo "Updated $ENV_FILE"
echo "Restarting llama-server (will download model if not cached)..."

sudo systemctl restart llama-server
sleep 5

if sudo systemctl is-active --quiet llama-server; then
  echo ""
  echo "=== Active model: $HF_FILE ==="
  echo "Health:  curl http://localhost:8080/health"
  echo "Metrics: curl http://localhost:8080/metrics | grep llama"
  echo "Logs:    sudo journalctl -u llama-server -f"
else
  echo "ERROR: llama-server failed to start"
  sudo journalctl -u llama-server -n 30
  exit 1
fi

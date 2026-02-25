#!/bin/bash

# Setup llama.cpp on Ubuntu (pedrogpt) with CUDA support.
# Builds llama-server and installs it as a systemd service that
# exposes Prometheus metrics at :8080/metrics.
#
# Usage: ./setup-llama-cpp-ubuntu.sh [--rebuild]
#   --rebuild  Force a clean rebuild even if llama.cpp is already installed

set -euo pipefail

LLAMA_DIR="/opt/llama.cpp"
MODEL_DIR="/opt/models"
ENV_FILE="/etc/llama-server.env"
SERVICE_FILE="/etc/systemd/system/llama-server.service"
PORT=8080

REBUILD=false
for arg in "$@"; do
  [[ "$arg" == "--rebuild" ]] && REBUILD=true
done

echo "=== llama.cpp Setup for pedrogpt ==="
echo "LLAMA_DIR:  $LLAMA_DIR"
echo "MODEL_DIR:  $MODEL_DIR"
echo "Port:       $PORT"
echo ""

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------
echo "--- Installing build dependencies ---"
sudo apt-get update -qq
sudo apt-get install -y \
  build-essential cmake git curl wget unzip \
  libssl-dev libcurl4-openssl-dev \
  python3-pip

# 1Password CLI (used by switch-model.sh to fetch HF_TOKEN)
if ! command -v op &>/dev/null; then
  echo "--- Installing 1Password CLI ---"
  OP_VERSION="2.30.0"
  curl -sSfLo /tmp/op.zip \
    "https://cache.agilebits.com/dist/1P/op2/pkg/v${OP_VERSION}/op_linux_amd64_v${OP_VERSION}.zip"
  sudo unzip -o /tmp/op.zip -d /usr/local/bin op
  sudo chmod +x /usr/local/bin/op
  rm /tmp/op.zip
  echo "op CLI installed: $(op --version)"
else
  echo "op CLI already installed: $(op --version)"
fi

# Prompt for OP_SERVICE_ACCOUNT_TOKEN if not already persisted
OP_ENV_FILE="/etc/op-service-account.env"
if [[ ! -f "$OP_ENV_FILE" ]]; then
  echo ""
  echo "--- 1Password service account setup ---"
  echo "Enter OP_SERVICE_ACCOUNT_TOKEN for pedrogpt (stored in $OP_ENV_FILE):"
  read -rsp "Token: " OP_TOKEN
  echo ""
  echo "OP_SERVICE_ACCOUNT_TOKEN=$OP_TOKEN" | sudo tee "$OP_ENV_FILE" > /dev/null
  sudo chmod 600 "$OP_ENV_FILE"
  echo "Token saved to $OP_ENV_FILE"
else
  echo "OP service account config already exists at $OP_ENV_FILE"
fi

# ---------------------------------------------------------------------------
# CUDA check — RTX 5090 (Blackwell, sm_120) requires CUDA 12.8+
# DO NOT use apt install nvidia-cuda-toolkit — it ships an old version.
# Install from NVIDIA's official repo:
#
#   wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb
#   sudo dpkg -i cuda-keyring_1.1-1_all.deb
#   sudo apt-get update
#   sudo apt-get install -y cuda-toolkit-12-8
#   echo 'export PATH=/usr/local/cuda/bin:$PATH' >> ~/.bashrc
#   source ~/.bashrc
#
# Then re-run this script.
# ---------------------------------------------------------------------------
CUDA_FLAG=""
CUDA_ARCH_FLAG=""

if ! command -v nvcc &>/dev/null; then
  echo ""
  echo "ERROR: nvcc not found — CUDA toolkit is not installed or not in PATH."
  echo ""
  echo "For RTX 5090 (Blackwell) install CUDA 12.8+ from NVIDIA's official repo:"
  echo "  wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb"
  echo "  sudo dpkg -i cuda-keyring_1.1-1_all.deb"
  echo "  sudo apt-get update && sudo apt-get install -y cuda-toolkit-12-8"
  echo "  echo 'export PATH=/usr/local/cuda/bin:\$PATH' >> ~/.bashrc && source ~/.bashrc"
  echo ""
  echo "Then re-run this script. Aborting."
  exit 1
else
  CUDA_VERSION=$(nvcc --version | grep -oP 'release \K[0-9]+\.[0-9]+')
  CUDA_MAJOR=$(echo "$CUDA_VERSION" | cut -d. -f1)
  CUDA_MINOR=$(echo "$CUDA_VERSION" | cut -d. -f2)
  echo "CUDA $CUDA_VERSION detected"

  # RTX 5090 (Blackwell sm_120) requires CUDA 12.8+
  if [[ "$CUDA_MAJOR" -lt 12 ]] || [[ "$CUDA_MAJOR" -eq 12 && "$CUDA_MINOR" -lt 8 ]]; then
    echo ""
    echo "ERROR: CUDA $CUDA_VERSION is too old for RTX 5090 (Blackwell)."
    echo "Blackwell (sm_120) requires CUDA 12.8 or newer."
    echo ""
    echo "Upgrade:"
    echo "  sudo apt-get install -y cuda-toolkit-12-8"
    echo "  (add NVIDIA's repo first if not already done — see above)"
    echo ""
    echo "Aborting."
    exit 1
  fi

  echo "CUDA $CUDA_VERSION OK — building with GGML_CUDA=ON, sm_120 (RTX 5090 Blackwell)"
  CUDA_FLAG="-DGGML_CUDA=ON"
  CUDA_ARCH_FLAG="-DCMAKE_CUDA_ARCHITECTURES=120"
fi

# ---------------------------------------------------------------------------
# Clone or update llama.cpp
# ---------------------------------------------------------------------------
if [[ -d "$LLAMA_DIR" && "$REBUILD" == "false" ]]; then
  echo "--- Updating existing llama.cpp clone ---"
  sudo git -C "$LLAMA_DIR" fetch origin
  sudo git -C "$LLAMA_DIR" reset --hard origin/master
else
  echo "--- Cloning llama.cpp ---"
  sudo rm -rf "$LLAMA_DIR"
  sudo git clone https://github.com/ggml-org/llama.cpp "$LLAMA_DIR"
fi

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
echo "--- Building llama.cpp (this takes a few minutes) ---"
cd "$LLAMA_DIR"

sudo cmake -B build \
  ${CUDA_FLAG} \
  ${CUDA_ARCH_FLAG} \
  -DCMAKE_BUILD_TYPE=Release \
  -DLLAMA_SERVER_VERBOSE=OFF

sudo cmake --build build --config Release -j "$(nproc)"

echo "--- Build complete ---"
ls -lh "$LLAMA_DIR/build/bin/"

# ---------------------------------------------------------------------------
# Model directory
# ---------------------------------------------------------------------------
sudo mkdir -p "$MODEL_DIR"
echo "Models go in: $MODEL_DIR"
echo "Use ./switch-model.sh to download and activate a model."

# ---------------------------------------------------------------------------
# Environment config (editable without touching the service file)
# ---------------------------------------------------------------------------
if [[ ! -f "$ENV_FILE" ]]; then
  echo "--- Creating $ENV_FILE ---"
  sudo tee "$ENV_FILE" > /dev/null <<'EOF'
# llama-server runtime configuration
# Edit this file then: sudo systemctl restart llama-server
# Switch models with: ./switch-model.sh <hf-repo> <hf-file>

# HuggingFace repo and file (llama-server downloads and caches automatically)
HF_REPO=unsloth/gpt-oss-20b-GGUF
HF_FILE=gpt-oss-20b-Q4_K_M.gguf

# HuggingFace cache directory (on the 2TB drive)
HF_HOME=/opt/models/cache

# GPU layers (-1 = all layers on GPU)
N_GPU_LAYERS=-1

# Context window size (tokens)
N_CTX=8192

# Number of parallel request slots
N_PARALLEL=4

# Server port (Tailscale serve maps this to HTTPS)
PORT=8080

# HuggingFace token (required for model downloads — keep this file root-only)
HF_TOKEN=your_token_here
EOF
  echo "Edit $ENV_FILE before starting the service."
else
  echo "--- $ENV_FILE already exists, skipping ---"
fi

# ---------------------------------------------------------------------------
# Systemd service
# ---------------------------------------------------------------------------
echo "--- Installing systemd service ---"
sudo tee "$SERVICE_FILE" > /dev/null <<EOF
[Unit]
Description=llama.cpp Server
Documentation=https://github.com/ggml-org/llama.cpp
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$LLAMA_DIR/build/bin/llama-server \\
    --host 0.0.0.0 \\
    --port \${PORT} \\
    --hf-repo \${HF_REPO} \\
    --hf-file \${HF_FILE} \\
    --ctx-size \${N_CTX} \\
    --n-gpu-layers \${N_GPU_LAYERS} \\
    --parallel \${N_PARALLEL} \\
    --jinja \\
    --no-webui \\
    --metrics
Environment=HF_HOME=\${HF_HOME}
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=llama-server
SupplementaryGroups=render video

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable llama-server

echo ""
echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "  1. Download all pedro models (~51 GB to /opt/models):"
echo "       ./switch-model.sh download-all"
echo ""
echo "  2. Start the server:"
echo "       sudo systemctl start llama-server"
echo "       sudo systemctl status llama-server"
echo ""
echo "  3. Verify metrics:"
echo "       curl http://localhost:8080/metrics | head -20"
echo ""
echo "  Logs: sudo journalctl -u llama-server -f"

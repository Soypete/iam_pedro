# Embeddings sidecar: llama.cpp server serving nomic-embed-text.
# Uses official llama.cpp Docker image with llama-server.

FROM ghcr.io/ggml-org/llama.cpp:server

# Download the embedding model
WORKDIR /models
ARG MODEL_URL=https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf
RUN curl -fSL "$MODEL_URL" -o nomic-embed-text.gguf \
    && test "$(head -c 4 nomic-embed-text.gguf)" = "GGUF" || (echo "ERROR: not a GGUF file" && exit 1)

EXPOSE 8081

# --ubatch-size >= --batch-size is required for non-causal embedding models.
ENTRYPOINT ["llama-server", \
  "--host", "0.0.0.0", "--port", "8081", \
  "-m", "/models/nomic-embed-text.gguf", \
  "--alias", "nomic-embed-text", \
  "--embeddings", "--pooling", "mean", \
  "--ctx-size", "2048", "--batch-size", "2048", "--ubatch-size", "2048", \
  "--metrics", "--no-webui"]

# Embeddings sidecar: a CPU-only llama.cpp server serving nomic-embed-text.
#
# Runs alongside the twitch bot in the same pod and exposes an OpenAI-compatible
# /v1/embeddings endpoint on :8081. The chat server (pedrogpt) runs MTP, which is
# incompatible with the embeddings graph, so embeddings live here instead.
#
# nomic-embed-text-v1.5 is 768-dim, mean-pooled. Tiny (~140MB at Q8_0) and fast on CPU.

# ---- build llama.cpp (CPU) ----
FROM ubuntu:24.04 AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential cmake git curl libcurl4-openssl-dev ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN git clone --depth 1 https://github.com/ggml-org/llama.cpp .
# CPU-only build (no CUDA): just llama-server.
RUN cmake -B build -DCMAKE_BUILD_TYPE=Release -DLLAMA_CURL=ON \
    && cmake --build build --config Release -j"$(nproc)" --target llama-server

# ---- download the embedding model ----
FROM ubuntu:24.04 AS model
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /models
# nomic-embed-text-v1.5 GGUF, Q8_0 (verified: GGUF magic, 146,146,432 bytes ≈ 139MB).
# Override MODEL_URL at build time to use a different quant/repo.
ARG MODEL_URL=https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf
RUN curl -fSL "$MODEL_URL" -o nomic-embed-text.gguf \
    && test "$(head -c 4 nomic-embed-text.gguf)" = "GGUF" || (echo "ERROR: not a GGUF file" && exit 1)

# ---- runtime ----
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
      libcurl4-openssl-dev libgomp1 ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /src/build/bin/llama-server /usr/local/bin/llama-server
COPY --from=model /models/nomic-embed-text.gguf /models/nomic-embed-text.gguf

EXPOSE 8081

# --ubatch-size >= --batch-size is required for non-causal embedding models.
ENTRYPOINT ["/usr/local/bin/llama-server", \
  "--host", "0.0.0.0", "--port", "8081", \
  "-m", "/models/nomic-embed-text.gguf", \
  "--alias", "nomic-embed-text", \
  "--embeddings", "--pooling", "mean", \
  "--ctx-size", "2048", "--batch-size", "2048", "--ubatch-size", "2048", \
  "--metrics", "--no-webui"]

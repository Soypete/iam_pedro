# Embeddings sidecar

A CPU-only llama.cpp server serving **nomic-embed-text** (768-dim, mean pooling) on `:8081`,
run as a sidecar in the twitch-bot pod. mem-palace and FAQ reach it via `http://localhost:8081`.

This exists because the pedrogpt chat server runs Qwen3.6-27B **MTP**, which is incompatible with
the embeddings graph (crashes on load) — so embeddings can't share that process.

## Files

- `Dockerfile` — builds CPU llama.cpp + bakes in the nomic GGUF (verified at build time).
- `build-embed.sh` — build + push just the `pedro-embed` image.
- `deploy-embed.sh` — full deploy: build+push embed and twitch images, pin both tags in
  `charts/pedro-bots/values.yaml`, `helm upgrade pedro -n chatbot`, and verify.

## Deploy

```bash
./deploy-embed.sh                 # build+push both images, bump tags, helm upgrade, verify
./deploy-embed.sh --no-deploy     # build+push + bump values only (review, deploy by hand)
./deploy-embed.sh --embed-only    # rebuild just the embed image
```

No secrets are handled here. The twitch/postgres creds come from OpenBAO injection in the chart;
the embed sidecar needs none. The pgvector migration (`0010_embeddings_768.sql`) runs automatically
at twitch-bot startup via goose — no manual migration step.

## Verify

```bash
POD=$(kubectl get pods -n chatbot -l app.kubernetes.io/component=twitch -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n chatbot "$POD" -c embed -- \
  curl -s localhost:8081/v1/embeddings -d '{"model":"nomic-embed-text","input":"hello"}'
kubectl logs -n chatbot "$POD" -c twitch-bot | grep -i ontology   # mem-palace loaded via :8081
```

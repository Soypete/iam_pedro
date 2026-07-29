# AGENTS.md

High-signal guidance for agents working in this repo.

## Build & Deploy

- **Always build for amd64**: Kubernetes cluster is amd64. Use `podman build --platform=linux/amd64`
- **Image location**: ZOT registry at `100.81.89.62:5000/`
- **Helm deploy**: `helm upgrade pedro charts/pedro-bots -n chatbot`
- **Check pods**: `kubectl get pods -n chatbot`

## Helm Template Pitfalls

The twitch-deployment.yaml has a subtle bug pattern: when `openbaoInjection: true`, the template sets `command` and `args`, but a second `args` block (lines 80-85) can override it. Always verify the rendered template with:
```bash
helm template pedro charts/pedro-bots -n chatbot | grep -A20 "twitch-bot"
```

## Vault/OpenBao Integration

- Service accounts must be added to Vault kubernetes auth role: `vault write auth/kubernetes/role/pedro bound_service_account_names=pedro-twitch,pedro-discord bound_service_account_namespaces=chatbot policies=pedro-all ttl=1h`
- Foundary stack.yaml supports `k8s_auth_roles` config for openbao component
- Vault secrets injected as files in `/vault/secrets/` must be sourced in container command: `. /vault/secrets/twitch && . /vault/secrets/postgres && exec /app/twitch/twitch <args>`

## Architecture

- Two independent bots: `cli/discord` and `cli/twitch` (separate k8s pods)
- Uses langchain-go connecting to local llama.cpp server (not OpenAI)
- EMBEDDINGS_PATH points to `pedro-embeddings` service: `http://pedro-embeddings.chatbot.svc.cluster.local:8081/v1`

## Testing

- Run all tests: `go test ./... -v -cover -covermode=atomic`
- Test files are untracked in git (`.go` files ending in `_test.go` throughout)
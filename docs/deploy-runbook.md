# Bot Deployment Runbook

This guide covers how to deploy services to the K3s cluster using Helm with OpenBAO secrets injection.

## Prerequisites

- Access to the K3s cluster via Foundry
- OpenBAO running at `100.81.89.62:8200`
- Zot registry at `100.81.89.62:5000`
- kubectl configured with Foundry kubeconfig

```bash
export KUBECONFIG=~/.foundry/kubeconfig
```

## Cluster Info

| Service | Address |
|---------|---------|
| OpenBAO | 100.81.89.62:8200 |
| Zot Registry | 100.81.89.62:5000 |
| K8s API | 100.81.89.100:6443 |

### Nodes
- **blue1** (100.81.89.62): Control plane + infrastructure
- **blue2** (100.70.90.12): Worker with 2TB storage
- **refurb** (100.125.196.1): Worker

---

## Adding a New Bot/Service

### Step 1: Create OpenBAO Secrets

1. Login to OpenBAO:
   ```bash
   export VAULT_ADDR=http://100.81.89.62:8200
   export VAULT_TOKEN=s.8sy7M9skEVO47Gsn12BtBjgO
   ```

2. Enable KV secrets engine if needed:
   ```bash
   vault secrets enable -path=pedro kv-v2
   ```

3. Write secrets:
   ```bash
   vault kv put pedro/discord \
     DISCORD_SECRET="your_secret_here" \
     DISCORD_CLIENT_ID="your_client_id" \
     DISCORD_PUBLIC_KEY="your_public_key" \
     POSTGRES_URL="postgresql://user:pass@host:5432/db?sslmode=require"
   ```

### Step 2: Create Kubernetes Service Account

Create a ServiceAccount in the target namespace that matches the OpenBAO role:

```yaml
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pedro-<service-name>
  namespace: chatbot
```

Apply:
```bash
kubectl apply -f serviceaccount.yaml
```

### Step 3: Update OpenBAO Role

The OpenBAO role must allow the service account. Update the auth role:

```bash
vault write auth/kubernetes/role/pedro-bots \
    bound_service_account_names="pedro-discord,pedro-twitch,pedro-keepalive,pedro-mempalace" \
    bound_service_account_namespaces=chatbot \
    policies=pedro-bots-policy \
    ttl=24h
```

### Step 4: Add Vault Annotations to Helm Template

In your deployment template (`templates/<service>-deployment.yaml`):

```yaml
metadata:
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "pedro-bots"
    vault.hashicorp.com/agent-inject-secret-{secret-name}: "pedro/{secret-path}"
    vault.hashicorp.com/agent-inject-template-{secret-name}: |
      {{- with secret "pedro/{secret-path}" -}}
      {{- range $k, $v := .data.data }}
      export {{ $k }}="{{ $v }}"
      {{- end }}
      {{- end }}
```

### Step 5: Update values.yaml

Enable OpenBAO injection:

```yaml
discord:
  openbaoInjection: true
  # ... other config
```

### Step 6: Deploy

```bash
cd charts/pedro-bots
helm upgrade pedro . -n chatbot
```

### Step 7: Verify

Check pods are running:
```bash
kubectl get pods -n chatbot -l app.kubernetes.io/name=pedro-bots
```

Verify secrets are injected:
```bash
kubectl exec -n chatbot <pod-name> -c <container> -- cat /vault/secrets/<secret-name>
```

---

## Deployment Commands

### Deploy to production
```bash
cd charts/pedro-bots
helm upgrade pedro . -n chatbot
```

### Restart a specific bot
```bash
kubectl rollout restart deployment/pedro-discord -n chatbot
```

### View logs
```bash
kubectl logs -n chatbot -l app.kubernetes.io/name=pedro-bots -f
```

### Rollback
```bash
helm rollback pedro -n chatbot
```

---

## Troubleshooting

### Secrets not injecting
1. Check ServiceAccount exists: `kubectl get sa pedro-<service> -n chatbot`
2. Verify vault annotations on pod: `kubectl describe pod <pod> | grep -A5 vault`
3. Check OpenBAO role: `vault read auth/kubernetes/role/pedro-bots`
4. Verify secrets exist: `vault kv get pedro/<service>`

### Pod not scheduling
1. Check node resources: `kubectl top nodes`
2. Review pod events: `kubectl describe pod <pod>`
3. Check nodeSelector/tolerations in values.yaml

### Image pull failures
1. Verify image exists in Zot: `curl http://100.81.89.62:5000/v2/_catalog`
2. Check image tag is correct in values.yaml
3. Ensure you're building for linux/amd64

---

## Common Issues

| Issue | Solution |
|-------|----------|
| Vault secrets show empty | Ensure POSTGRES_URL is in vault KV under data.data |
| Pod crashloop | Check command path - use `/usr/local/bin/<wrapper>` |
| Image "latest" missing binaries | Use specific image tags (commit hash) |
| High CPU on control-plane | Remove nodeSelector to allow worker scheduling |

---

## Useful Commands

```bash
# Get all pods with node info
kubectl get pods -n chatbot -o wide

# Check OpenBAO status
vault status -address=http://100.81.89.62:8200

# List all vault secrets
vault kv list pedro/

# Check pod vault sidecar
kubectl get pod <pod> -n chatbot -o jsonpath='{.spec.containers[*].name}'
```
#!/bin/bash

# deploy.sh - Deploy IAM_PEDRO bots using Helm
set -e

# Configuration
NAMESPACE="pedro-bots"
RELEASE_NAME="pedro"
CHART_PATH="./charts/pedro-discord"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if kubectl is available and connected
check_kubernetes() {
    print_status "Checking Kubernetes connection..."
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl not found. Please install kubectl first."
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    print_success "Connected to Kubernetes cluster"
}

# Function to create namespace if it doesn't exist
create_namespace() {
    print_status "Creating namespace '$NAMESPACE' if it doesn't exist..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    print_success "Namespace '$NAMESPACE' ready"
}

# Function to load secrets from 1Password
load_secrets() {
    print_status "Loading secrets from 1Password..."
    
    if ! command -v op &> /dev/null; then
        print_error "1Password CLI (op) not found. Please install it first."
        print_error "Visit: https://developer.1password.com/docs/cli/get-started/"
        exit 1
    fi
    
    # Check if signed in to 1Password
    if ! op account list &> /dev/null; then
        print_error "Not signed in to 1Password. Please run 'op signin' first."
        exit 1
    fi
    
    # Source the prod.env file and resolve 1Password references
    print_status "Resolving 1Password secrets from prod.env..."
    
    # Create temporary values file with secrets
    local temp_values=$(mktemp)
    cat > "$temp_values" <<EOF
secrets:
EOF
    
    # Read prod.env and resolve secrets
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "$line" ]] && continue
        
        # Extract variable name and 1Password reference
        if [[ "$line" =~ ^([^=]+)=\"?([^\"]+)\"?$ ]]; then
            var_name="${BASH_REMATCH[1]}"
            op_ref="${BASH_REMATCH[2]}"
            
            # Resolve 1Password reference
            if [[ "$op_ref" =~ ^op:// ]]; then
                print_status "Resolving $var_name..."
                secret_value=$(op read "$op_ref" 2>/dev/null || echo "")
                
                if [[ -n "$secret_value" ]]; then
                    # Convert environment variable names to values.yaml format
                    case "$var_name" in
                        "TWITCH_SECRET") echo "  twitchSecret: \"$secret_value\"" >> "$temp_values" ;;
                        "POSTGRES_URL") echo "  postgresUrl: \"$secret_value\"" >> "$temp_values" ;;
                        "POSTGRES_VECTOR_URL") echo "  postgresVectorUrl: \"$secret_value\"" >> "$temp_values" ;;
                        "DISCORD_SECRET") echo "  discordSecret: \"$secret_value\"" >> "$temp_values" ;;
                        "DISCORD_CLIENT_ID") echo "  discordClientId: \"$secret_value\"" >> "$temp_values" ;;
                        "DISCORD_PUBLIC_KEY") echo "  discordPublicKey: \"$secret_value\"" >> "$temp_values" ;;
                        "DISCORD_PERMISSION") echo "  discordPermission: \"$secret_value\"" >> "$temp_values" ;;
                        "SUPABASE_PUB_KEY") echo "  supabasePubKey: \"$secret_value\"" >> "$temp_values" ;;
                        "SUPABASE_PRIV_KEY") echo "  supabasePrivKey: \"$secret_value\"" >> "$temp_values" ;;
                        "SUBAPASE_URL") echo "  supabaseUrl: \"$secret_value\"" >> "$temp_values" ;;
                        "SUBAPASE_JWT") echo "  supabaseJwt: \"$secret_value\"" >> "$temp_values" ;;
                        "LLAMA_CPP_PATH") echo "  llamaCppPath: \"$secret_value\"" >> "$temp_values" ;;
                    esac
                else
                    print_warning "Could not resolve $var_name from 1Password"
                fi
            fi
        fi
    done < prod.env
    
    echo "$temp_values"
}

# Function to deploy using Helm
deploy_helm() {
    print_status "Deploying IAM_PEDRO bots using Helm..."
    
    if ! command -v helm &> /dev/null; then
        print_error "Helm not found. Please install Helm first."
        exit 1
    fi
    
    # Check if chart exists
    if [[ ! -d "$CHART_PATH" ]]; then
        print_error "Helm chart not found at $CHART_PATH"
        exit 1
    fi
    
    # Load secrets from 1Password
    local secrets_file
    secrets_file=$(load_secrets)
    
    # Deploy or upgrade
    if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
        print_status "Upgrading existing release '$RELEASE_NAME'..."
        helm upgrade "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$secrets_file" \
            --wait \
            --timeout=300s
    else
        print_status "Installing new release '$RELEASE_NAME'..."
        helm install "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --create-namespace \
            --values "$secrets_file" \
            --wait \
            --timeout=300s
    fi
    
    # Clean up temporary secrets file
    rm -f "$secrets_file"
    
    print_success "Helm deployment completed"
}

# Function to show deployment status
show_status() {
    print_status "Checking deployment status..."
    echo
    echo "=== Release Status ==="
    helm status "$RELEASE_NAME" -n "$NAMESPACE"
    echo
    echo "=== Pod Status ==="
    kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/instance=$RELEASE_NAME"
    echo
    echo "=== Service Status ==="
    kubectl get services -n "$NAMESPACE" -l "app.kubernetes.io/instance=$RELEASE_NAME"
}

# Function to show logs
show_logs() {
    local component=$1
    if [[ -z "$component" ]]; then
        print_error "Please specify component: discord or twitch"
        return 1
    fi
    
    print_status "Showing logs for $component bot..."
    kubectl logs -n "$NAMESPACE" -l "app.kubernetes.io/component=$component" --tail=50 -f
}

# Function to delete deployment
delete_deployment() {
    print_warning "This will delete the entire IAM_PEDRO deployment!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Deleting Helm release '$RELEASE_NAME'..."
        helm uninstall "$RELEASE_NAME" -n "$NAMESPACE"
        print_success "Deployment deleted"
    else
        print_status "Deletion cancelled"
    fi
}

# Main script logic
case "${1:-deploy}" in
    "deploy")
        check_kubernetes
        create_namespace
        deploy_helm
        show_status
        ;;
    "status")
        show_status
        ;;
    "logs")
        show_logs "$2"
        ;;
    "delete")
        delete_deployment
        ;;
    "help")
        echo "Usage: $0 [command] [options]"
        echo
        echo "Commands:"
        echo "  deploy    Deploy or upgrade the bots (default)"
        echo "  status    Show deployment status"
        echo "  logs      Show logs for a component (discord|twitch)"
        echo "  delete    Delete the deployment"
        echo "  help      Show this help message"
        echo
        echo "Examples:"
        echo "  $0 deploy              # Deploy the bots"
        echo "  $0 status              # Check status"
        echo "  $0 logs discord        # Show Discord bot logs"
        echo "  $0 logs twitch         # Show Twitch bot logs"
        echo "  $0 delete              # Delete deployment"
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
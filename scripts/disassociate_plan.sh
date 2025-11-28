#!/bin/bash

# Script to disassociate a subscription plan from a node group

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${AUTH_TOKEN}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

print_success() {
    echo -e "${GREEN}SUCCESS: $1${NC}"
}

print_info() {
    echo -e "${YELLOW}INFO: $1${NC}"
}

# Check if required arguments are provided
if [ -z "$1" ] || [ -z "$2" ]; then
    print_error "Usage: $0 <group_id> <plan_id>"
    echo ""
    echo "Example:"
    echo "  $0 1 2"
    echo ""
    echo "Environment variables:"
    echo "  AUTH_TOKEN  - Authentication token (required)"
    echo "  BASE_URL    - API base URL (default: http://localhost:8080)"
    exit 1
fi

GROUP_ID=$1
PLAN_ID=$2

# Check if token is set
if [ -z "$TOKEN" ]; then
    print_error "AUTH_TOKEN environment variable is not set"
    echo "Please set it using: export AUTH_TOKEN=your_token_here"
    exit 1
fi

print_info "Checking current associations for node group ${GROUP_ID}..."

# First, check if the group is associated with this plan
CHECK_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET \
    "${BASE_URL}/node-groups/${GROUP_ID}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json")

CHECK_HTTP_CODE=$(echo "$CHECK_RESPONSE" | tail -n1)
CHECK_BODY=$(echo "$CHECK_RESPONSE" | sed '$d')

if [ "$CHECK_HTTP_CODE" != "200" ]; then
    print_error "Failed to fetch node group (HTTP $CHECK_HTTP_CODE)"
    echo "$CHECK_BODY" | jq -r '.message // .error // .' 2>/dev/null || echo "$CHECK_BODY"
    exit 1
fi

# Check if plan is associated
IS_ASSOCIATED=$(echo "$CHECK_BODY" | jq -r ".data.subscription_plan_ids[]? | select(. == $PLAN_ID)" 2>/dev/null)

if [ -z "$IS_ASSOCIATED" ]; then
    print_error "Node group ${GROUP_ID} is NOT associated with plan ${PLAN_ID}"
    echo ""
    echo "Current associated plan IDs:"
    CURRENT_PLANS=$(echo "$CHECK_BODY" | jq -r '.data.subscription_plan_ids[]?' 2>/dev/null)
    if [ -z "$CURRENT_PLANS" ]; then
        echo "  (none)"
    else
        echo "$CURRENT_PLANS" | while read PID; do
            echo "  - Plan ID: $PID"
        done
    fi
    exit 1
fi

print_success "Confirmed: Node group ${GROUP_ID} is associated with plan ${PLAN_ID}"
print_info "Disassociating plan ${PLAN_ID} from node group ${GROUP_ID}..."

# Disassociate the plan
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
    "${BASE_URL}/node-groups/${GROUP_ID}/plans/${PLAN_ID}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json")

# Extract HTTP status code and body
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" == "204" ]; then
    print_success "Plan ${PLAN_ID} successfully disassociated from node group ${GROUP_ID}"
    exit 0
else
    print_error "Failed to disassociate plan (HTTP $HTTP_CODE)"
    if [ -n "$BODY" ]; then
        echo "$BODY" | jq -r '.message // .error // .' 2>/dev/null || echo "$BODY"
    fi
    exit 1
fi

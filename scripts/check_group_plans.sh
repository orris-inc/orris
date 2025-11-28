#!/bin/bash

# Script to check which subscription plans are associated with a node group

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

# Check if GROUP_ID is provided
if [ -z "$1" ]; then
    print_error "Usage: $0 <group_id>"
    exit 1
fi

GROUP_ID=$1

# Check if token is set
if [ -z "$TOKEN" ]; then
    print_error "AUTH_TOKEN environment variable is not set"
    echo "Please set it using: export AUTH_TOKEN=your_token_here"
    exit 1
fi

print_info "Fetching details for node group ${GROUP_ID}..."

# Get node group details
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET \
    "${BASE_URL}/node-groups/${GROUP_ID}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json")

# Extract HTTP status code and body
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    print_error "Failed to fetch node group (HTTP $HTTP_CODE)"
    echo "$BODY" | jq -r '.message // .error // .' 2>/dev/null || echo "$BODY"
    exit 1
fi

print_success "Node group found"
echo ""

# Parse and display node group information
echo "Node Group Details:"
echo "==================="
echo "$BODY" | jq -r '.data |
    "ID: \(.id)",
    "Name: \(.name)",
    "Description: \(.description // "N/A")",
    "Is Public: \(.is_public)",
    "Sort Order: \(.sort_order)"' 2>/dev/null

echo ""
echo "Associated Subscription Plan IDs:"
echo "=================================="

PLAN_IDS=$(echo "$BODY" | jq -r '.data.subscription_plan_ids[]?' 2>/dev/null)

if [ -z "$PLAN_IDS" ]; then
    print_info "No subscription plans associated with this node group"
else
    echo "$PLAN_IDS" | while read PLAN_ID; do
        echo "  - Plan ID: $PLAN_ID"
    done
fi

echo ""
echo "Node IDs in Group:"
echo "=================="

NODE_IDS=$(echo "$BODY" | jq -r '.data.node_ids[]?' 2>/dev/null)

if [ -z "$NODE_IDS" ]; then
    print_info "No nodes in this group"
else
    NODE_COUNT=$(echo "$NODE_IDS" | wc -l | tr -d ' ')
    echo "  Total nodes: $NODE_COUNT"
    echo "$NODE_IDS" | while read NODE_ID; do
        echo "  - Node ID: $NODE_ID"
    done
fi

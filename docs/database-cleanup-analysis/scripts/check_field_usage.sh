#!/bin/bash
# check_field_usage.sh - Check field usage across the codebase
# Usage: ./check_field_usage.sh <field_name> [table_name]

set -e

FIELD=$1
TABLE=${2:-""}

if [ -z "$FIELD" ]; then
    echo "Usage: $0 <field_name> [table_name]"
    echo "Example: $0 APIRequests subscription_usages"
    exit 1
fi

PROJECT_ROOT="/Users/easayliu/Documents/go/orris"
cd "$PROJECT_ROOT"

echo "========================================"
echo "Checking usage for field: $FIELD"
if [ -n "$TABLE" ]; then
    echo "In table: $TABLE"
fi
echo "========================================"
echo ""

# Function to search and display results
search_layer() {
    local layer=$1
    local path=$2
    local pattern=$3
    
    echo "=== $layer ==="
    local result=$(grep -rn "$pattern" "$path" 2>/dev/null | grep -v "Binary file" || true)
    
    if [ -z "$result" ]; then
        echo "✅ No references found"
    else
        echo "❌ Found references:"
        echo "$result" | head -20
        local count=$(echo "$result" | wc -l | tr -d ' ')
        if [ "$count" -gt 20 ]; then
            echo "... and $(($count - 20)) more"
        fi
    fi
    echo ""
}

# Search in different layers
search_layer "Model Layer" "internal/infrastructure/persistence/models/" "\b${FIELD}\b"
search_layer "Mapper Layer" "internal/infrastructure/persistence/mappers/" "\b${FIELD}\b"
search_layer "Domain Layer" "internal/domain/" "\b${FIELD}\b"
search_layer "Repository Layer" "internal/infrastructure/repository/" "\b${FIELD}\b"
search_layer "Use Case Layer" "internal/application/" "\b${FIELD}\b"
search_layer "DTO Layer" "internal/application/" "\b${FIELD}\b"
search_layer "Handler Layer" "internal/interfaces/http/handlers/" "\b${FIELD}\b"
search_layer "Middleware Layer" "internal/interfaces/http/middleware/" "\b${FIELD}\b"

# Search in migration scripts
echo "=== Migration Scripts ==="
local migration_result=$(grep -rn "$FIELD" "internal/infrastructure/migration/scripts/" 2>/dev/null | grep -v "Binary file" || true)
if [ -z "$migration_result" ]; then
    echo "✅ No migration scripts found"
else
    echo "Found in migrations:"
    echo "$migration_result"
fi
echo ""

# Search in database column format (snake_case)
SNAKE_CASE=$(echo "$FIELD" | sed 's/\([A-Z]\)/_\L\1/g' | sed 's/^_//')
if [ "$SNAKE_CASE" != "$FIELD" ]; then
    echo "=== Also checking snake_case: $SNAKE_CASE ==="
    search_layer "All Go files" "internal/" "\b${SNAKE_CASE}\b"
fi

# Summary
echo "========================================"
echo "Summary"
echo "========================================"
total_count=$(grep -r "\b${FIELD}\b" internal/ 2>/dev/null | grep -v "Binary file" | wc -l | tr -d ' ')
echo "Total references found: $total_count"

if [ "$total_count" -eq 0 ]; then
    echo "✅ SAFE TO DELETE - No references found in codebase"
elif [ "$total_count" -lt 10 ]; then
    echo "⚠️  LOW USAGE - Few references found, review them carefully"
else
    echo "❌ HIGH USAGE - Many references found, deletion will require significant refactoring"
fi

echo ""
echo "To see all occurrences:"
echo "  grep -rn '\b${FIELD}\b' internal/"

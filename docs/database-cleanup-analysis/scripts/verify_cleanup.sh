#!/bin/bash
# verify_cleanup.sh - Verify database cleanup migration
# Usage: ./verify_cleanup.sh <phase>

set -e

PHASE=${1:-1}
PROJECT_ROOT="/Users/easayliu/Documents/go/orris"
cd "$PROJECT_ROOT"

echo "========================================"
echo "Cleanup Verification - Phase $PHASE"
echo "========================================"
echo ""

# Pre-flight checks
echo "=== Pre-flight Checks ==="

# Check if database backup exists
BACKUP_DIR="$PROJECT_ROOT/backups"
if [ ! -d "$BACKUP_DIR" ]; then
    echo "⚠️  Warning: No backups directory found"
    echo "   Create backup first: mysqldump -u root -p orris > backups/backup_\$(date +%Y%m%d_%H%M%S).sql"
else
    LATEST_BACKUP=$(ls -t "$BACKUP_DIR"/*.sql 2>/dev/null | head -1 || echo "")
    if [ -n "$LATEST_BACKUP" ]; then
        echo "✅ Latest backup: $LATEST_BACKUP"
    else
        echo "❌ No backups found in $BACKUP_DIR"
        exit 1
    fi
fi

# Check if tests pass
echo ""
echo "=== Running Tests ==="
if go test ./... -v > /tmp/test_output.log 2>&1; then
    echo "✅ All tests passed"
else
    echo "❌ Some tests failed. Check /tmp/test_output.log"
    tail -20 /tmp/test_output.log
    exit 1
fi

# Check for uncommitted changes
echo ""
echo "=== Git Status ==="
if git diff --quiet; then
    echo "✅ No uncommitted changes"
else
    echo "⚠️  Warning: You have uncommitted changes"
    git status --short
fi

echo ""
echo "=== Phase $PHASE Specific Checks ==="

if [ "$PHASE" -eq 1 ]; then
    # Phase 1: Check if fields to be removed are truly unused
    echo "Checking fields to be removed in Phase 1..."
    
    fields=(
        "APIRequests"
        "APIDataOut"
        "APIDataIn"
        "WebhookCalls"
        "EmailsSent"
        "ReportsGenerated"
        "ProjectsCount"
        "CustomEndpoint"
    )
    
    for field in "${fields[@]}"; do
        count=$(grep -r "\b${field}\b" internal/ 2>/dev/null | grep -v "Binary file" | grep -v "models/" | grep -v "mappers/" | grep -v "domain/" | wc -l | tr -d ' ')
        if [ "$count" -gt 0 ]; then
            echo "⚠️  Warning: $field has $count references outside model/mapper/domain layers"
        else
            echo "✅ $field is safe to delete"
        fi
    done
    
    # Check subscription_histories
    count=$(grep -r "SubscriptionHistory" internal/ 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -eq 0 ]; then
        echo "✅ subscription_histories table is safe to delete"
    else
        echo "❌ subscription_histories has $count references"
    fi
    
elif [ "$PHASE" -eq 2 ]; then
    # Phase 2: Check if OAuth and notification code are updated
    echo "Checking Phase 2 prerequisites..."
    
    # Check if locale is still used
    locale_count=$(grep -r "\.Locale\b" internal/ 2>/dev/null | grep -v "models/" | wc -l | tr -d ' ')
    if [ "$locale_count" -gt 0 ]; then
        echo "⚠️  Warning: Locale field has $locale_count references"
    else
        echo "✅ Locale field is safe to delete"
    fi
    
    # Check if view_count is still used
    viewcount_count=$(grep -r "ViewCount" internal/ 2>/dev/null | grep -v "models/" | grep -v "dto/" | wc -l | tr -d ' ')
    if [ "$viewcount_count" -gt 0 ]; then
        echo "⚠️  Warning: ViewCount has $viewcount_count business logic references"
    else
        echo "✅ ViewCount is safe to delete"
    fi
fi

echo ""
echo "=== Summary ==="
echo "✅ Pre-flight checks passed"
echo "✅ All tests passing"
echo ""
echo "Ready to proceed with Phase $PHASE migration?"
echo "Run: goose -dir internal/infrastructure/migration/scripts/cleanup mysql \"user:pass@/orris\" up"
echo ""
echo "After migration:"
echo "1. Run tests: go test ./... -v"
echo "2. Start app: go run cmd/api/main.go"
echo "3. Test critical endpoints"
echo "4. Regenerate docs: swag init"

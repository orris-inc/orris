# Database Cleanup Scripts

This directory contains utility scripts for the database cleanup project.

## Available Scripts

### 1. check_field_usage.sh
**Purpose**: Check how a database field is used across the codebase

**Usage**:
```bash
./check_field_usage.sh <field_name> [table_name]
```

**Examples**:
```bash
# Check usage of APIRequests field
./check_field_usage.sh APIRequests subscription_usages

# Check usage of CustomEndpoint (auto-searches all tables)
./check_field_usage.sh CustomEndpoint

# Check usage of ViewCount
./check_field_usage.sh ViewCount announcements
```

**Output**:
- Model layer references
- Mapper layer references
- Domain layer references
- Repository layer references
- Use case layer references
- Handler layer references
- Migration script references
- Total count and risk assessment

### 2. verify_cleanup.sh
**Purpose**: Verify prerequisites before running cleanup migration

**Usage**:
```bash
./verify_cleanup.sh <phase>
```

**Examples**:
```bash
# Verify Phase 1 prerequisites
./verify_cleanup.sh 1

# Verify Phase 2 prerequisites
./verify_cleanup.sh 2
```

**Checks**:
- Database backup exists
- All tests passing
- No uncommitted changes
- Fields to be removed are truly unused
- Special phase-specific checks

## Prerequisites

### System Requirements
- Bash 4.0+
- MySQL client
- Go 1.21+
- Git

### Environment Setup
```bash
# Make scripts executable
chmod +x check_field_usage.sh
chmod +x verify_cleanup.sh

# Verify environment
go version
mysql --version
git --version
```

## Common Workflows

### Before Running Phase 1 Migration

```bash
# 1. Check each field to be removed
./check_field_usage.sh APIRequests
./check_field_usage.sh APIDataOut
./check_field_usage.sh CustomEndpoint

# 2. Verify prerequisites
./verify_cleanup.sh 1

# 3. If all checks pass, proceed with migration
cd ../internal/infrastructure/migration/scripts/cleanup
goose mysql "user:pass@/orris" up-to 8
```

### Before Running Phase 2 Migration

```bash
# 1. Ensure Phase 1 is complete and stable
./verify_cleanup.sh 1

# 2. Check Phase 2 fields
./check_field_usage.sh Locale users
./check_field_usage.sh ViewCount announcements

# 3. Verify Phase 2 prerequisites
./verify_cleanup.sh 2

# 4. Proceed with migration
cd ../internal/infrastructure/migration/scripts/cleanup
goose mysql "user:pass@/orris" up-to 9
```

## Troubleshooting

### Script Permission Denied
```bash
chmod +x *.sh
```

### MySQL Connection Issues
```bash
# Test connection
mysql -u root -p orris -e "SELECT 1;"
```

### False Positives in Field Usage Check
The script may report references in:
- Comments
- Test files
- Generated documentation

Manually review the output to filter these out.

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Verify Database Cleanup

on:
  pull_request:
    paths:
      - 'internal/infrastructure/migration/scripts/cleanup/**'

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run verification
        run: |
          cd scripts
          ./verify_cleanup.sh 1
```

## Contributing

When adding new scripts:
1. Follow the naming convention: `<action>_<target>.sh`
2. Add usage documentation to this README
3. Include error handling and user-friendly output
4. Make scripts idempotent when possible

## Related Documentation

- [Database Cleanup Index](../DATABASE_CLEANUP_INDEX.md)
- [Quick Reference Guide](../CLEANUP_QUICK_REFERENCE.md)
- [Execution Summary](../CLEANUP_EXECUTION_SUMMARY.md)

# Database Cleanup Migration Scripts

## Overview

This directory contains migration scripts for removing redundant and unused database fields from the Orris project.

## Migration Files

### 008_phase1_remove_unused_fields.sql
**Risk Level**: ✅ Zero Risk  
**Estimated Time**: 2-3 hours  
**Impact**:
- Removes `subscription_histories` table (never implemented)
- Removes 7 unused fields from `subscription_usages` table
- Removes `custom_endpoint` from `subscription_plans` table

**Code Changes Required**:
- ~500 lines to remove (Model, Mapper, Domain layers)
- No business logic changes needed

### 009_phase2_remove_low_usage_fields.sql
**Risk Level**: 🟡 Low Risk  
**Estimated Time**: 4-5 hours  
**Impact**:
- Removes `locale` from `users` table
- Removes `view_count` from `announcements` table  
- Removes `archived_at` from `notifications` table

**Code Changes Required**:
- ~200 lines to remove
- OAuth integration needs update (2 files)
- Notification archive logic needs update

## Execution Order

```bash
# 1. Backup database
mysqldump -u root -p orris > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. Run Phase 1 (zero risk)
goose -dir internal/infrastructure/migration/scripts/cleanup mysql "user:pass@/orris" up-to 8

# 3. Verify and test
go test ./... -v

# 4. Run Phase 2 (after code cleanup)
goose -dir internal/infrastructure/migration/scripts/cleanup mysql "user:pass@/orris" up-to 9
```

## Code Cleanup Checklist

### For Each Removed Field

#### Phase 1 - subscription_histories table
- [ ] No code to clean (table never implemented)

#### Phase 1 - subscription_usages fields
For each field: `api_requests`, `api_data_out`, `api_data_in`, `webhook_calls`, `emails_sent`, `reports_generated`, `projects_count`

Files to modify:
- [ ] `internal/infrastructure/persistence/models/subscriptionusagemodel.go` - Remove field
- [ ] `internal/infrastructure/persistence/mappers/subscriptionusagemapper.go` - Remove mapping
- [ ] `internal/domain/subscription/subscriptionusage.go` - Remove field, getter, setter
- [ ] `internal/infrastructure/repository/subscriptionusagerepository.go` - Review queries

#### Phase 1 - subscription_plans.custom_endpoint
- [ ] `internal/infrastructure/persistence/models/subscriptionplanmodel.go`
- [ ] `internal/infrastructure/persistence/mappers/subscriptionplanmapper.go`
- [ ] `internal/domain/subscription/subscriptionplan.go`
- [ ] `internal/application/subscription/dto/dto.go`

#### Phase 2 - users.locale
- [ ] `internal/infrastructure/persistence/models/usermodel.go`
- [ ] `internal/infrastructure/auth/oauthgoogle.go` - Remove locale setting

#### Phase 2 - announcements.view_count
- [ ] `internal/infrastructure/persistence/models/announcementmodel.go`
- [ ] `internal/domain/notification/announcement.go` - Remove IncrementViewCount()
- [ ] `internal/application/notification/usecases/getannouncement.go` - Remove increment call
- [ ] `internal/interfaces/dto/notificationdto.go` - Remove view_count field

#### Phase 2 - notifications.archived_at
- [ ] `internal/infrastructure/persistence/models/notificationmodel.go`
- [ ] `internal/infrastructure/persistence/mappers/notificationmapper.go` - Remove special mapping logic
- [ ] `internal/domain/notification/notification.go` - Use DeletedAt instead

## Verification Steps

After each phase:

```bash
# 1. Check database schema
mysql -u root -p orris -e "DESCRIBE subscription_usages;"
mysql -u root -p orris -e "SHOW TABLES;" | grep subscription_histories

# 2. Run tests
go test ./internal/infrastructure/repository/... -v
go test ./internal/application/... -v
go test ./internal/interfaces/... -v

# 3. Regenerate Swagger docs
swag init

# 4. Start application and verify
go run cmd/api/main.go

# 5. Test critical APIs
# - User login (OAuth)
# - Subscription creation
# - Node management
# - Announcement viewing
```

## Rollback Procedure

If issues are found:

```bash
# Rollback to specific version
goose -dir internal/infrastructure/migration/scripts/cleanup mysql "user:pass@/orris" down-to 7

# Or restore from backup
mysql -u root -p orris < backup_20251112_xxxxx.sql
```

## Performance Impact

**Expected Improvements**:
- Database size: -10-15%
- Query performance: +2-5% (fewer columns to scan)
- Backup time: -10%

**No Negative Impact Expected**:
- All removed fields are unused or rarely used
- No impact on critical business logic

## Documentation Updates

After migration:
- [ ] Update API documentation
- [ ] Update Swagger annotations
- [ ] Update database schema diagram
- [ ] Update README if needed

## Contact

For questions or issues:
- Review the main report: `DATABASE_REDUNDANCY_ANALYSIS_REPORT.md`
- Check quick reference: `CLEANUP_QUICK_REFERENCE.md`

---

**Created**: 2025-11-12  
**Last Updated**: 2025-11-12

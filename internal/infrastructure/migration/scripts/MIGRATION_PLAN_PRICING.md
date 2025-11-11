# Subscription Plan Pricing Migration

## Overview
This migration adds support for multiple pricing options per subscription plan, allowing different prices for different billing cycles (weekly, monthly, quarterly, semi-annual, yearly, lifetime).

## Migration: 005_subscription_plan_pricing.sql

### What it does
1. Creates a new table `subscription_plan_pricing`
2. Migrates existing pricing data from `subscription_plans` table
3. Establishes foreign key relationships with cascade delete

### Database Changes

#### New Table: subscription_plan_pricing
```sql
Columns:
- id: BIGINT UNSIGNED (Primary Key)
- plan_id: BIGINT UNSIGNED (Foreign Key → subscription_plans.id)
- billing_cycle: VARCHAR(20) (weekly|monthly|quarterly|semi_annual|yearly|lifetime)
- price: BIGINT UNSIGNED (in cents)
- currency: VARCHAR(3) (CNY|USD|EUR|GBP|JPY)
- is_active: BOOLEAN (default TRUE)
- created_at: TIMESTAMP
- updated_at: TIMESTAMP
- deleted_at: TIMESTAMP (soft delete)

Constraints:
- UNIQUE (plan_id, billing_cycle): Each plan can only have one price per billing cycle
- FK plan_id: CASCADE DELETE when plan is deleted

Indexes:
- idx_plan_id: Fast lookup by plan
- idx_billing_cycle: Filter by billing cycle
- idx_is_active: Filter active pricings
- idx_deleted_at: Soft delete support
```

### Data Migration
The migration automatically copies existing data:
- Source: `subscription_plans.price`, `subscription_plans.billing_cycle`
- Target: `subscription_plan_pricing` table
- Only active plans with valid price and billing_cycle are migrated

### Impact on Existing System

#### Backward Compatibility
- ✅ `subscription_plans` table keeps original `price` and `billing_cycle` columns
- ✅ Existing queries continue to work
- ✅ New API endpoints are additive, not breaking

#### Forward Migration Steps
1. Run migration: `goose up`
2. Deploy application code with new pricing logic
3. Test multi-pricing functionality
4. Gradually update admin UI to manage multiple pricings

### Rollback Strategy

#### Option 1: Database Rollback
```bash
goose down
```
This will:
- Drop `subscription_plan_pricing` table
- Restore system to pre-migration state

#### Option 2: Keep Data, Disable Feature
- Don't delete the table
- Application falls back to single pricing model
- Data preserved for future use

### Testing Checklist

#### Before Migration
- [ ] Backup production database
- [ ] Test migration on staging environment
- [ ] Verify data count: `SELECT COUNT(*) FROM subscription_plans WHERE price > 0`

#### After Migration
- [ ] Verify pricing count: `SELECT COUNT(*) FROM subscription_plan_pricing`
- [ ] Check foreign key integrity: `SELECT * FROM subscription_plan_pricing WHERE plan_id NOT IN (SELECT id FROM subscription_plans)`
- [ ] Test cascade delete: Create test plan → Add pricings → Delete plan → Verify pricings deleted

#### Application Testing
- [ ] GET /subscription-plans/public returns pricings array
- [ ] POST /subscriptions accepts billing_cycle parameter
- [ ] Admin can add/update/delete pricing options
- [ ] Price changes reflect immediately in API

### Example Queries

#### Get all pricings for a plan
```sql
SELECT * FROM subscription_plan_pricing
WHERE plan_id = ? AND is_active = TRUE AND deleted_at IS NULL
ORDER BY FIELD(billing_cycle, 'weekly', 'monthly', 'quarterly', 'semi_annual', 'yearly', 'lifetime');
```

#### Get specific pricing
```sql
SELECT * FROM subscription_plan_pricing
WHERE plan_id = ? AND billing_cycle = ? AND deleted_at IS NULL;
```

#### Check pricing gaps (plans without pricing)
```sql
SELECT id, name, slug
FROM subscription_plans
WHERE id NOT IN (
    SELECT DISTINCT plan_id FROM subscription_plan_pricing WHERE deleted_at IS NULL
)
AND deleted_at IS NULL;
```

### Performance Considerations
- **Index Coverage**: All common queries use indexes
- **Join Performance**: Foreign key relationship is indexed
- **Query Pattern**: Typically 1-6 pricings per plan (low cardinality)
- **Estimated Row Size**: ~100 bytes per pricing record

### Security Considerations
- **Price Integrity**: BIGINT UNSIGNED prevents negative prices
- **Currency Validation**: Limited to 3-character codes
- **Soft Delete**: Preserves audit trail
- **Cascade Delete**: Maintains referential integrity

### Post-Migration Tasks
1. Update monitoring dashboards to track pricing table metrics
2. Add alerts for plans with missing pricings
3. Document new pricing management workflows
4. Train support team on multi-pricing feature

### Troubleshooting

#### Issue: Migration fails on data copy
**Cause**: NULL values in price or billing_cycle
**Solution**: Clean data before migration
```sql
-- Check problematic records
SELECT id, name, price, billing_cycle FROM subscription_plans
WHERE (price IS NULL OR price = 0 OR billing_cycle IS NULL) AND deleted_at IS NULL;
```

#### Issue: Duplicate key error on unique constraint
**Cause**: Multiple plans with same (plan_id, billing_cycle) in source data
**Solution**: Deduplicate source data
```sql
-- Find duplicates
SELECT plan_id, billing_cycle, COUNT(*)
FROM subscription_plan_pricing
GROUP BY plan_id, billing_cycle
HAVING COUNT(*) > 1;
```

### Related Files
- Domain: `/internal/domain/subscription/value_objects/planpricing.go`
- Model: `/internal/infrastructure/persistence/models/planpricingmodel.go`
- Repository: `/internal/infrastructure/repository/planpricingrepository.go`
- UseCase: `/internal/application/subscription/usecases/getplanpricings.go`

### Version Information
- Migration Version: 005
- Created: 2025-01-10
- Database: MySQL 5.7+ / MariaDB 10.2+
- Go Version: 1.21+

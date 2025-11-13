# PlanFeatures Traffic Limit Configuration Guide

## Overview

The `PlanFeatures` value object in the Subscription domain provides comprehensive support for traffic and resource limit configuration. This guide demonstrates how to properly configure and use traffic limits within subscription plans.

## Configuration Verification

### Current Structure

The `PlanFeatures` struct supports flexible limit storage:

```go
type PlanFeatures struct {
    Features []string               `json:"features"`
    Limits   map[string]interface{} `json:"limits"`
}
```

### Standard Limit Keys

The following standard keys are defined as constants for consistency:

| Constant | Key | Type | Description |
|----------|-----|------|-------------|
| `LimitKeyTraffic` | `"traffic_limit"` | `uint64` | Monthly traffic limit in bytes (0 = unlimited) |
| `LimitKeyDeviceCount` | `"device_limit"` | `int` | Maximum concurrent devices (0 = unlimited) |
| `LimitKeySpeedLimit` | `"speed_limit"` | `int` | Speed limit in Mbps (0 = unlimited) |
| `LimitKeyConnectionLimit` | `"connection_limit"` | `int` | Maximum concurrent connections (0 = unlimited) |
| `LimitKeyNodeAccess` | `"node_access"` | `[]uint` | Accessible node group IDs (empty = all accessible) |

## Usage Examples

### 1. Creating a Plan with Traffic Limits

#### Basic Plan (100GB/month)

```go
package main

import (
    "orris/internal/domain/subscription"
    vo "orris/internal/domain/subscription/value_objects"
)

func createBasicPlan() (*subscription.SubscriptionPlan, error) {
    // Create the base plan
    plan, err := subscription.NewSubscriptionPlan(
        "Basic Plan",
        "basic",
        "Perfect for individual users",
        999,  // 9.99 USD in cents
        "USD",
        vo.BillingCycleMonthly,
        7, // 7 days trial
    )
    if err != nil {
        return nil, err
    }

    // Configure features and limits
    features := vo.NewPlanFeatures(
        []string{
            "basic_support",
            "email_notifications",
        },
        map[string]interface{}{
            vo.LimitKeyTraffic:     uint64(100 * 1024 * 1024 * 1024), // 100GB
            vo.LimitKeyDeviceCount: 3,                                 // 3 devices
            vo.LimitKeySpeedLimit:  100,                               // 100 Mbps
        },
    )

    // Apply features to plan
    if err := plan.UpdateFeatures(features); err != nil {
        return nil, err
    }

    return plan, nil
}
```

#### Premium Plan (Unlimited Traffic)

```go
func createPremiumPlan() (*subscription.SubscriptionPlan, error) {
    plan, err := subscription.NewSubscriptionPlan(
        "Premium Plan",
        "premium",
        "Unlimited everything for power users",
        4999, // 49.99 USD in cents
        "USD",
        vo.BillingCycleMonthly,
        14, // 14 days trial
    )
    if err != nil {
        return nil, err
    }

    features := vo.NewPlanFeatures(
        []string{
            "24_7_support",
            "advanced_analytics",
            "api_access",
        },
        map[string]interface{}{
            vo.LimitKeyTraffic:         uint64(0), // Unlimited
            vo.LimitKeyDeviceCount:     10,
            vo.LimitKeySpeedLimit:      1000, // 1 Gbps
            vo.LimitKeyConnectionLimit: 500,
        },
    )

    if err := plan.UpdateFeatures(features); err != nil {
        return nil, err
    }

    return plan, nil
}
```

### 2. Using Typed Setter Methods (Recommended)

```go
func configureTrafficLimits() *vo.PlanFeatures {
    pf := vo.NewPlanFeatures(nil, nil)

    // Use type-safe setters (recommended approach)
    pf.SetTrafficLimit(500 * 1024 * 1024 * 1024) // 500GB
    pf.SetDeviceLimit(5)
    pf.SetSpeedLimit(200) // 200 Mbps
    pf.SetConnectionLimit(100)
    pf.SetNodeAccess([]uint{1, 2, 3})

    return pf
}
```

### 3. Checking Traffic Limits

#### Direct Plan Methods

```go
func checkPlanTrafficLimit(plan *subscription.SubscriptionPlan) {
    // Check if plan has unlimited traffic
    if plan.IsUnlimitedTraffic() {
        fmt.Println("Plan has unlimited traffic")
        return
    }

    // Get traffic limit
    limit, err := plan.GetTrafficLimit()
    if err != nil {
        log.Printf("Error getting traffic limit: %v", err)
        return
    }

    // Convert bytes to GB for display
    limitGB := float64(limit) / (1024 * 1024 * 1024)
    fmt.Printf("Monthly traffic limit: %.2f GB\n", limitGB)

    // Check if user has remaining traffic
    usedBytes := uint64(150 * 1024 * 1024 * 1024) // 150GB used
    hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
    if err != nil {
        log.Printf("Error checking traffic: %v", err)
        return
    }

    if !hasRemaining {
        fmt.Println("User has exceeded traffic limit")
    } else {
        fmt.Println("User has remaining traffic")
    }
}
```

#### PlanFeatures Methods

```go
func checkFeatureTrafficLimit(features *vo.PlanFeatures) {
    // Get traffic limit from features
    limit, err := features.GetTrafficLimit()
    if err != nil {
        log.Printf("Error: %v", err)
        return
    }

    if limit == 0 {
        fmt.Println("Unlimited traffic")
    } else {
        limitGB := float64(limit) / (1024 * 1024 * 1024)
        fmt.Printf("Traffic limit: %.2f GB\n", limitGB)
    }

    // Validate traffic usage
    usedBytes := uint64(200 * 1024 * 1024 * 1024) // 200GB
    hasRemaining, err := features.HasTrafficRemaining(usedBytes)
    if err != nil {
        log.Printf("Error: %v", err)
        return
    }

    if hasRemaining {
        fmt.Println("Within traffic limit")
    } else {
        fmt.Println("Exceeded traffic limit")
    }
}
```

### 4. Use Case Integration Example

```go
package usecases

import (
    "context"
    "fmt"

    "orris/internal/domain/subscription"
)

type ValidateTrafficUsageUseCase struct {
    planRepo subscription.SubscriptionPlanRepository
}

func (uc *ValidateTrafficUsageUseCase) Execute(
    ctx context.Context,
    userID uint,
    usedBytes uint64,
) (bool, error) {
    // Get user's subscription plan
    plan, err := uc.planRepo.FindByUserID(ctx, userID)
    if err != nil {
        return false, fmt.Errorf("failed to get user plan: %w", err)
    }

    // Check if plan allows current traffic usage
    hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
    if err != nil {
        return false, fmt.Errorf("failed to check traffic limit: %w", err)
    }

    return hasRemaining, nil
}
```

### 5. Traffic Unit Conversion Helpers

```go
package helpers

const (
    ByteInKB = 1024
    ByteInMB = 1024 * ByteInKB
    ByteInGB = 1024 * ByteInMB
    ByteInTB = 1024 * ByteInGB
)

func GBToBytes(gb int) uint64 {
    return uint64(gb) * ByteInGB
}

func BytesToGB(bytes uint64) float64 {
    return float64(bytes) / ByteInGB
}

func MBToBytes(mb int) uint64 {
    return uint64(mb) * ByteInMB
}

// Usage example
func createPlanWithTrafficLimit(gb int) map[string]interface{} {
    return map[string]interface{}{
        vo.LimitKeyTraffic: GBToBytes(gb),
    }
}
```

### 6. Complete UseCase Example: Creating Plan via API

```go
package usecases

import (
    "context"
    "fmt"

    "orris/internal/application/subscription/dto"
    "orris/internal/domain/subscription"
    vo "orris/internal/domain/subscription/value_objects"
)

type CreateSubscriptionPlanCommand struct {
    Name           string
    Slug           string
    Description    string
    Price          uint64
    Currency       string
    BillingCycle   string
    TrialDays      int
    Features       []string
    TrafficLimitGB int  // Traffic limit in GB (0 = unlimited)
    DeviceLimit    int  // Device limit (0 = unlimited)
    SpeedLimitMbps int  // Speed limit in Mbps (0 = unlimited)
}

type CreateSubscriptionPlanUseCase struct {
    planRepo subscription.SubscriptionPlanRepository
}

func (uc *CreateSubscriptionPlanUseCase) Execute(
    ctx context.Context,
    cmd CreateSubscriptionPlanCommand,
) (*dto.SubscriptionPlanDTO, error) {
    // Parse billing cycle
    billingCycle, err := vo.NewBillingCycle(cmd.BillingCycle)
    if err != nil {
        return nil, fmt.Errorf("invalid billing cycle: %w", err)
    }

    // Create plan
    plan, err := subscription.NewSubscriptionPlan(
        cmd.Name,
        cmd.Slug,
        cmd.Description,
        cmd.Price,
        cmd.Currency,
        *billingCycle,
        cmd.TrialDays,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create plan: %w", err)
    }

    // Configure features and limits
    features := vo.NewPlanFeatures(cmd.Features, nil)

    // Set traffic limit (convert GB to bytes)
    if cmd.TrafficLimitGB > 0 {
        trafficBytes := uint64(cmd.TrafficLimitGB) * 1024 * 1024 * 1024
        features.SetTrafficLimit(trafficBytes)
    } else {
        features.SetTrafficLimit(0) // unlimited
    }

    // Set device limit
    if cmd.DeviceLimit > 0 {
        if err := features.SetDeviceLimit(cmd.DeviceLimit); err != nil {
            return nil, err
        }
    }

    // Set speed limit
    if cmd.SpeedLimitMbps > 0 {
        if err := features.SetSpeedLimit(cmd.SpeedLimitMbps); err != nil {
            return nil, err
        }
    }

    // Apply features to plan
    if err := plan.UpdateFeatures(features); err != nil {
        return nil, fmt.Errorf("failed to update features: %w", err)
    }

    // Persist plan
    if err := uc.planRepo.Create(ctx, plan); err != nil {
        return nil, fmt.Errorf("failed to persist plan: %w", err)
    }

    // Return DTO
    return uc.toDTO(plan), nil
}

func (uc *CreateSubscriptionPlanUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
    dto := &dto.SubscriptionPlanDTO{
        ID:           plan.ID(),
        Name:         plan.Name(),
        Slug:         plan.Slug(),
        Description:  plan.Description(),
        Price:        plan.Price(),
        Currency:     plan.Currency(),
        BillingCycle: plan.BillingCycle().String(),
        TrialDays:    plan.TrialDays(),
        Status:       string(plan.Status()),
        CreatedAt:    plan.CreatedAt(),
        UpdatedAt:    plan.UpdatedAt(),
    }

    if features := plan.Features(); features != nil {
        dto.Features = features.Features
        dto.Limits = features.Limits
    }

    return dto
}
```

## API Request/Response Examples

### Creating a Plan with Traffic Limits

**Request:**
```json
POST /api/v1/subscription-plans

{
  "name": "Standard Plan",
  "slug": "standard",
  "description": "Perfect for small teams",
  "price": 2999,
  "currency": "USD",
  "billing_cycle": "monthly",
  "trial_days": 14,
  "features": [
    "priority_support",
    "advanced_analytics"
  ],
  "limits": {
    "traffic_limit": 536870912000,
    "device_limit": 5,
    "speed_limit": 500,
    "connection_limit": 100,
    "node_access": [1, 2, 3]
  }
}
```

**Response:**
```json
{
  "id": 1,
  "name": "Standard Plan",
  "slug": "standard",
  "description": "Perfect for small teams",
  "price": 2999,
  "currency": "USD",
  "billing_cycle": "monthly",
  "trial_days": 14,
  "status": "active",
  "features": [
    "priority_support",
    "advanced_analytics"
  ],
  "limits": {
    "traffic_limit": 536870912000,
    "device_limit": 5,
    "speed_limit": 500,
    "connection_limit": 100,
    "node_access": [1, 2, 3]
  },
  "created_at": "2025-11-12T10:00:00Z",
  "updated_at": "2025-11-12T10:00:00Z"
}
```

## Best Practices

### 1. Use Typed Methods
Always prefer typed getter/setter methods over direct map access:
```go
// ✅ Good
features.SetTrafficLimit(100 * GB)
limit, err := features.GetTrafficLimit()

// ❌ Avoid
features.SetLimit("traffic_limit", 100 * GB)
limit, ok := features.GetLimit("traffic_limit")
```

### 2. Use Constants for Keys
Always use predefined constants for limit keys:
```go
// ✅ Good
features.SetLimit(vo.LimitKeyTraffic, bytes)

// ❌ Avoid
features.SetLimit("traffic_limit", bytes)
```

### 3. Handle Zero Values Consistently
0 always means unlimited:
```go
// Unlimited traffic
features.SetTrafficLimit(0)

// 100GB limit
features.SetTrafficLimit(100 * GB)
```

### 4. Validate Before Setting
Always validate input before setting limits:
```go
if trafficGB < 0 {
    return fmt.Errorf("traffic limit cannot be negative")
}
features.SetTrafficLimit(uint64(trafficGB) * GB)
```

### 5. Use Helper Functions for Conversions
Create helper functions for common conversions:
```go
func TrafficGBToBytes(gb int) uint64 {
    return uint64(gb) * 1024 * 1024 * 1024
}

func TrafficBytesToGB(bytes uint64) float64 {
    return float64(bytes) / (1024 * 1024 * 1024)
}
```

## Testing Examples

```go
package value_objects_test

import (
    "testing"

    vo "orris/internal/domain/subscription/value_objects"
)

func TestTrafficLimit(t *testing.T) {
    t.Run("unlimited traffic", func(t *testing.T) {
        pf := vo.NewPlanFeatures(nil, nil)
        pf.SetTrafficLimit(0)

        if !pf.IsUnlimitedTraffic() {
            t.Error("expected unlimited traffic")
        }

        // Any usage should be within limit
        hasRemaining, err := pf.HasTrafficRemaining(1000000000000)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !hasRemaining {
            t.Error("unlimited plan should always have remaining traffic")
        }
    })

    t.Run("limited traffic", func(t *testing.T) {
        pf := vo.NewPlanFeatures(nil, nil)
        limit := uint64(100 * 1024 * 1024 * 1024) // 100GB
        pf.SetTrafficLimit(limit)

        // Under limit
        hasRemaining, _ := pf.HasTrafficRemaining(50 * 1024 * 1024 * 1024)
        if !hasRemaining {
            t.Error("50GB should be under 100GB limit")
        }

        // Over limit
        hasRemaining, _ = pf.HasTrafficRemaining(150 * 1024 * 1024 * 1024)
        if hasRemaining {
            t.Error("150GB should exceed 100GB limit")
        }
    })
}
```

## Summary

### Verification Results
✅ **PlanFeatures.Limits can store traffic limit configuration**
- The `Limits map[string]interface{}` field supports flexible limit storage
- Type-safe getter/setter methods are provided

✅ **Standard limit key naming is defined**
- `LimitKeyTraffic = "traffic_limit"` (monthly traffic in bytes)
- `LimitKeyDeviceCount = "device_limit"` (concurrent devices)
- `LimitKeySpeedLimit = "speed_limit"` (speed in Mbps)
- `LimitKeyConnectionLimit = "connection_limit"` (concurrent connections)
- `LimitKeyNodeAccess = "node_access"` (accessible node group IDs)

✅ **Convenient methods for traffic limits**
- `GetTrafficLimit()` - Get traffic limit in bytes
- `SetTrafficLimit()` - Set traffic limit in bytes
- `IsUnlimitedTraffic()` - Check if traffic is unlimited
- `HasTrafficRemaining()` - Validate traffic usage

✅ **SubscriptionPlan convenience methods**
- `GetTrafficLimit()` - Direct access to traffic limit
- `IsUnlimitedTraffic()` - Check unlimited status
- `HasTrafficRemaining()` - Validate usage against limit

### Improvements Made
1. Added standard limit key constants for consistency
2. Implemented type-safe getter/setter methods for all limit types
3. Added convenience methods for traffic validation
4. Provided comprehensive usage examples
5. Included API request/response examples
6. Defined best practices and testing patterns

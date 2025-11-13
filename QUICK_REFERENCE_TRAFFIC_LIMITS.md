# 流量限制配置快速参考

## 标准限制键常量

```go
vo.LimitKeyTraffic         // "traffic_limit"      - 月流量限制(字节)
vo.LimitKeyDeviceCount     // "device_limit"       - 并发设备数
vo.LimitKeySpeedLimit      // "speed_limit"        - 速度限制(Mbps)
vo.LimitKeyConnectionLimit // "connection_limit"   - 并发连接数
vo.LimitKeyNodeAccess      // "node_access"        - 节点组ID列表
```

## 单位转换常量

```go
const (
    KB = 1024
    MB = 1024 * KB          // 1,048,576
    GB = 1024 * MB          // 1,073,741,824
    TB = 1024 * GB          // 1,099,511,627,776
)

// 示例
traffic100GB := uint64(100 * GB)  // 107,374,182,400 bytes
traffic500GB := uint64(500 * GB)  // 536,870,912,000 bytes
traffic1TB := uint64(1 * TB)      // 1,099,511,627,776 bytes
```

## 快速创建示例

### 基础套餐 (100GB)
```go
features := vo.NewPlanFeatures(
    []string{"basic_support"},
    map[string]interface{}{
        vo.LimitKeyTraffic:     uint64(100 * GB),
        vo.LimitKeyDeviceCount: 3,
        vo.LimitKeySpeedLimit:  100,
    },
)
```

### 标准套餐 (500GB)
```go
features := vo.NewPlanFeatures(
    []string{"priority_support", "advanced_analytics"},
    map[string]interface{}{
        vo.LimitKeyTraffic:         uint64(500 * GB),
        vo.LimitKeyDeviceCount:     5,
        vo.LimitKeySpeedLimit:      500,
        vo.LimitKeyConnectionLimit: 100,
        vo.LimitKeyNodeAccess:      []uint{1, 2, 3},
    },
)
```

### 高级套餐 (无限流量)
```go
features := vo.NewPlanFeatures(
    []string{"24_7_support", "api_access"},
    map[string]interface{}{
        vo.LimitKeyTraffic:         uint64(0), // 0 = 无限制
        vo.LimitKeyDeviceCount:     10,
        vo.LimitKeySpeedLimit:      1000,
        vo.LimitKeyConnectionLimit: 500,
    },
)
```

## 类型安全的方法 (推荐)

### 设置限制
```go
features := vo.NewPlanFeatures(nil, nil)

features.SetTrafficLimit(500 * GB)           // 设置流量
features.SetDeviceLimit(5)                    // 设置设备数
features.SetSpeedLimit(200)                   // 设置速度(Mbps)
features.SetConnectionLimit(100)              // 设置连接数
features.SetNodeAccess([]uint{1, 2, 3})      // 设置节点组
```

### 获取限制
```go
// 流量限制
trafficLimit, err := features.GetTrafficLimit()
if err != nil { /* 处理错误 */ }

// 设备限制
deviceLimit, err := features.GetDeviceLimit()
if err != nil { /* 处理错误 */ }

// 速度限制
speedLimit, err := features.GetSpeedLimit()
if err != nil { /* 处理错误 */ }

// 连接限制
connLimit, err := features.GetConnectionLimit()
if err != nil { /* 处理错误 */ }

// 节点访问
nodeGroups, err := features.GetNodeAccess()
if err != nil { /* 处理错误 */ }
```

### 流量验证
```go
// 检查是否无限流量
if features.IsUnlimitedTraffic() {
    // 无限流量逻辑
}

// 检查是否超出流量
usedBytes := uint64(150 * GB)
hasRemaining, err := features.HasTrafficRemaining(usedBytes)
if err != nil { /* 处理错误 */ }

if !hasRemaining {
    // 用户已超出流量限制
}
```

## SubscriptionPlan 便捷方法

```go
// 直接从 Plan 获取流量限制
limit, err := plan.GetTrafficLimit()

// 检查是否无限流量
if plan.IsUnlimitedTraffic() {
    // 无限流量
}

// 验证流量使用
hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
```

## 常见模式

### 1. 创建Plan并设置流量限制
```go
plan, _ := subscription.NewSubscriptionPlan(
    "Basic Plan", "basic", "Description",
    999, "USD", vo.BillingCycleMonthly, 7,
)

features := vo.NewPlanFeatures(nil, nil)
features.SetTrafficLimit(100 * GB)
features.SetDeviceLimit(3)

plan.UpdateFeatures(features)
```

### 2. 验证用户流量使用
```go
func ValidateUserTraffic(plan *subscription.SubscriptionPlan, usedBytes uint64) error {
    hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
    if err != nil {
        return fmt.Errorf("failed to check traffic: %w", err)
    }

    if !hasRemaining {
        return fmt.Errorf("traffic limit exceeded")
    }

    return nil
}
```

### 3. 显示流量使用情况
```go
func DisplayTrafficUsage(features *vo.PlanFeatures, usedBytes uint64) {
    limit, _ := features.GetTrafficLimit()

    if limit == 0 {
        fmt.Println("Unlimited traffic")
        return
    }

    usedGB := float64(usedBytes) / float64(GB)
    limitGB := float64(limit) / float64(GB)
    percentage := (float64(usedBytes) / float64(limit)) * 100

    fmt.Printf("Used: %.2f GB / %.2f GB (%.1f%%)\n", usedGB, limitGB, percentage)
}
```

### 4. API请求示例
```bash
# 创建带流量限制的套餐
curl -X POST http://localhost:8080/api/v1/subscription-plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Standard Plan",
    "slug": "standard",
    "price": 2999,
    "currency": "USD",
    "billing_cycle": "monthly",
    "trial_days": 14,
    "features": ["priority_support"],
    "limits": {
      "traffic_limit": 536870912000,
      "device_limit": 5,
      "speed_limit": 500
    }
  }'
```

## 零值语义

| 限制类型 | 零值含义 |
|---------|---------|
| `traffic_limit: 0` | 无限流量 |
| `device_limit: 0` | 无限设备 |
| `speed_limit: 0` | 无限速度 |
| `connection_limit: 0` | 无限连接 |
| `node_access: []` | 所有节点可访问 |

## 错误处理

```go
// 所有 setter 方法都会验证负数
if err := features.SetDeviceLimit(-1); err != nil {
    // Error: device limit cannot be negative
}

// 类型转换错误
limit, err := features.GetTrafficLimit()
if err != nil {
    // 处理类型转换错误
}
```

## 测试覆盖

所有方法都有完整的单元测试:
```bash
go test -v ./internal/domain/subscription/value_objects/...
```

## 相关文件

- 实现: `/internal/domain/subscription/value_objects/planfeatures.go`
- 测试: `/internal/domain/subscription/value_objects/planfeatures_test.go`
- 示例: `/internal/domain/subscription/value_objects/planfeatures_example.go`
- 完整指南: `/PLAN_FEATURES_USAGE_GUIDE.md`
- 验证报告: `/PLAN_FEATURES_VERIFICATION_REPORT.md`

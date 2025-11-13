# Orris 系统并发安全问题深度分析报告

## 执行概要

本报告针对 Orris 代理节点管理系统进行了全面的并发安全分析。系统采用了乐观锁（Optimistic Locking）机制，但仍存在多个潜在的并发安全问题，包括竞态条件、幂等性缺失、流量统计不准确等。

**关键发现**: 共发现 **12 个并发安全问题**，其中：
- 🔴 **P0 级（严重）**: 4 个 - 必须立即修复
- 🟡 **P1 级（中等）**: 5 个 - 应该尽快修复
- 🟢 **P2 级（轻微）**: 3 个 - 可以后续优化

---

## 一、乐观锁实现检查

### ✅ 已正确实现乐观锁的 Repository

经过检查，以下 Repository 均正确实现了乐观锁机制：

1. **PaymentRepository** (`paymentrepository.go:38`)
   ```go
   Where("id = ? AND version = ?", model.ID, model.Version-1)
   ```

2. **TicketRepository** (`ticketrepository.go:43`)
   ```go
   Where("id = ? AND version = ?", model.ID, model.Version-1)
   ```

3. **UserRepositoryDDD** (`userrepositoryddd.go:131`)
   ```go
   Where("id = ? AND version = ?", model.ID, currentModel.Version)
   ```

4. **SubscriptionRepository** (`subscriptionrepository.go:191-193`)
   ```go
   previousVersion := model.Version - 1
   Where("id = ? AND version = ?", model.ID, previousVersion)
   ```

5. **NodeRepository** (`noderepository.go:116-118`)
   ```go
   previousVersion := model.Version - 1
   Where("id = ? AND version = ?", model.ID, previousVersion)
   ```

6. **NodeGroupRepository** (`nodegrouprepository.go:128-130`)
   ```go
   previousVersion := model.Version - 1
   Where("id = ? AND version = ?", model.ID, previousVersion)
   ```

**结论**: 所有核心 Repository 的乐观锁实现均符合规范。✅

---

## 二、发现的并发安全问题

### 🔴 P0-1: 支付回调处理缺少幂等性保护（严重）

**问题编号**: P0-1
**严重程度**: 🔴 严重
**问题类型**: 幂等性缺失
**受影响文件**: `/Users/easayliu/Documents/go/orris/internal/application/payment/usecases/handle_payment_callback.go:36-59`

#### 问题描述
支付网关可能发送重复回调通知，当前实现只在内存中检查支付状态，存在以下并发场景：

**并发场景**:
```
时间线:
T1: 回调请求 A 到达 -> GetByGatewayOrderNo() -> status=pending
T2: 回调请求 B 到达 -> GetByGatewayOrderNo() -> status=pending (仍然是 pending)
T3: 请求 A 执行 MarkAsPaid() -> Update() 成功
T4: 请求 B 执行 MarkAsPaid() -> Update() 成功 (乐观锁会失败，但逻辑不完善)
```

虽然乐观锁会阻止同一个 payment 被重复更新，但可能导致：
1. 订阅被重复激活（调用多次 `ActivateSubscription`）
2. 业务日志重复记录
3. 数据库负载增加

#### 当前实现（有问题）
```go
func (uc *HandlePaymentCallbackUseCase) Execute(ctx context.Context, req *http.Request) error {
    callbackData, err := uc.gateway.VerifyCallback(req)
    if err != nil {
        return fmt.Errorf("invalid callback: %w", err)
    }

    paymentOrder, err := uc.paymentRepo.GetByGatewayOrderNo(ctx, callbackData.GatewayOrderNo)
    if err != nil {
        return fmt.Errorf("payment not found: %w", err)
    }

    // 只检查内存状态，不够安全
    if paymentOrder.Status() == vo.PaymentStatusPaid {
        uc.logger.Infow("payment already processed", "payment_id", paymentOrder.ID())
        return nil
    }

    // ... 处理支付成功
}
```

#### 问题分析
1. **竞态条件**: 两个并发请求同时读取到 `status=pending`，都会尝试处理
2. **缺少事务级别的幂等性检查**: 没有基于 `transaction_id` 的唯一性约束
3. **订阅激活可能重复**: `ActivateSubscription` 可能被调用多次

#### 修复方案

**方案 1: 添加 TransactionID 唯一约束（推荐）**

1. 数据库层面添加唯一索引：
```sql
ALTER TABLE payments
ADD UNIQUE INDEX idx_transaction_id (transaction_id);
```

2. 修改代码逻辑：
```go
func (uc *HandlePaymentCallbackUseCase) Execute(ctx context.Context, req *http.Request) error {
    callbackData, err := uc.gateway.VerifyCallback(req)
    if err != nil {
        return fmt.Errorf("invalid callback: %w", err)
    }

    paymentOrder, err := uc.paymentRepo.GetByGatewayOrderNo(ctx, callbackData.GatewayOrderNo)
    if err != nil {
        return fmt.Errorf("payment not found: %w", err)
    }

    // 早期返回：已处理的支付
    if paymentOrder.Status() == vo.PaymentStatusPaid {
        uc.logger.Infow("payment already processed (idempotent check)",
            "payment_id", paymentOrder.ID(),
            "transaction_id", callbackData.TransactionID)
        return nil
    }

    if callbackData.Status == "TRADE_SUCCESS" || callbackData.Status == "success" {
        return uc.handlePaymentSuccessWithIdempotency(ctx, paymentOrder, callbackData)
    } else {
        return uc.handlePaymentFailure(ctx, paymentOrder, callbackData)
    }
}

func (uc *HandlePaymentCallbackUseCase) handlePaymentSuccessWithIdempotency(
    ctx context.Context,
    paymentOrder *payment.Payment,
    callbackData *payment_gateway.CallbackData,
) error {
    // 使用数据库事务确保原子性
    return uc.db.Transaction(func(tx *gorm.DB) error {
        // 1. 标记支付为已支付（包含 transaction_id）
        if err := paymentOrder.MarkAsPaid(callbackData.TransactionID); err != nil {
            return fmt.Errorf("failed to mark payment as paid: %w", err)
        }

        // 2. 更新支付记录（乐观锁 + transaction_id 唯一约束）
        if err := uc.paymentRepo.UpdateWithTx(ctx, tx, paymentOrder); err != nil {
            // 如果是唯一约束冲突，说明已经处理过了（幂等性）
            if strings.Contains(err.Error(), "idx_transaction_id") ||
               strings.Contains(err.Error(), "Duplicate entry") {
                uc.logger.Infow("payment already processed by another request (database constraint)",
                    "payment_id", paymentOrder.ID(),
                    "transaction_id", callbackData.TransactionID)
                return nil // 幂等性返回成功
            }
            return fmt.Errorf("failed to update payment: %w", err)
        }

        // 3. 激活订阅（只有在支付更新成功后才执行）
        activateCmd := subscriptionUsecases.ActivateSubscriptionCommand{
            SubscriptionID: paymentOrder.SubscriptionID(),
        }

        if err := uc.activateSubscriptionUC.Execute(ctx, activateCmd); err != nil {
            uc.logger.Errorw("failed to activate subscription",
                "error", err,
                "subscription_id", paymentOrder.SubscriptionID())
            return fmt.Errorf("failed to activate subscription: %w", err)
        }

        uc.logger.Infow("payment processed successfully",
            "payment_id", paymentOrder.ID(),
            "subscription_id", paymentOrder.SubscriptionID(),
            "transaction_id", callbackData.TransactionID)

        return nil
    })
}
```

**方案 2: 使用分布式锁（备选）**
```go
lockKey := fmt.Sprintf("payment:callback:%s", callbackData.GatewayOrderNo)
lock := redislock.Obtain(ctx, uc.redis, lockKey, 30*time.Second)
if lock == nil {
    uc.logger.Warnw("another request is processing this payment callback")
    return nil
}
defer lock.Release(ctx)

// 处理回调...
```

#### 复现概率
**高** - 支付网关通常会在 30 秒内重试多次，并发概率很高。

#### 潜在影响
- 订阅可能被重复激活（虽然订阅的 `Activate()` 方法有检查）
- 数据库乐观锁冲突，导致错误日志
- 可能误导运营人员

---

### 🔴 P0-2: 支付超时检查与回调并发冲突（严重）

**问题编号**: P0-2
**严重程度**: 🔴 严重
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/application/payment/usecases/expire_payments.go`
- `/Users/easayliu/Documents/go/orris/internal/infrastructure/scheduler/payment_scheduler.go`

#### 问题描述
定时任务每 5 分钟检查一次过期支付，可能与支付网关回调同时操作同一个 Payment 记录。

**并发场景**:
```
时间线:
T1: 支付创建，状态 = pending，expired_at = 15:00
T2 (15:01): 用户实际支付，但回调延迟
T3 (15:02): 定时任务执行 -> 查询到过期支付 -> MarkAsExpired()
T4 (15:03): 支付回调到达 -> MarkAsPaid()
```

**结果**: 取决于乐观锁的执行顺序：
- 如果定时任务先执行：支付被标记为 `expired`，回调会因乐观锁失败
- 如果回调先执行：支付正确标记为 `paid`，定时任务会跳过（状态已变）

虽然乐观锁能防止数据不一致，但会导致：
1. 用户已支付但订单显示过期（最严重）
2. 客服工单增加
3. 业务逻辑混乱

#### 当前实现（有问题）
```go
// expire_payments.go:26-70
func (uc *ExpirePaymentsUseCase) Execute(ctx context.Context) (int, error) {
    expiredPayments, err := uc.paymentRepo.GetExpiredPayments(ctx)
    if err != nil {
        return 0, fmt.Errorf("failed to get expired payments: %w", err)
    }

    expiredCount := 0
    for _, p := range expiredPayments {
        // 没有检查是否有pending的支付回调
        if err := p.MarkAsExpired(); err != nil {
            continue
        }

        if err := uc.paymentRepo.Update(ctx, p); err != nil {
            // 乐观锁失败会记录错误，但不会重试
            continue
        }
        expiredCount++
    }

    return expiredCount, nil
}
```

#### 问题分析
1. **时序竞争**: 超时检查与回调处理没有协调机制
2. **缺少状态二次确认**: 标记过期前没有再次检查支付网关状态
3. **用户体验差**: 用户支付成功却看到订单过期

#### 修复方案

**方案 1: 增加缓冲时间 + 二次确认（推荐）**

```go
func (uc *ExpirePaymentsUseCase) Execute(ctx context.Context) (int, error) {
    expiredPayments, err := uc.paymentRepo.GetExpiredPayments(ctx)
    if err != nil {
        return 0, fmt.Errorf("failed to get expired payments: %w", err)
    }

    if len(expiredPayments) == 0 {
        return 0, nil
    }

    uc.logger.Infow("processing expired payments", "count", len(expiredPayments))

    expiredCount := 0
    for _, p := range expiredPayments {
        // 安全措施1: 只处理过期超过5分钟的订单（给回调足够的缓冲时间）
        if time.Since(p.ExpiredAt()) < 5*time.Minute {
            uc.logger.Debugw("payment expired recently, skipping for safety",
                "payment_id", p.ID(),
                "expired_at", p.ExpiredAt())
            continue
        }

        // 安全措施2: 再次从数据库获取最新状态
        latestPayment, err := uc.paymentRepo.GetByID(ctx, p.ID())
        if err != nil {
            uc.logger.Errorw("failed to get latest payment status",
                "error", err,
                "payment_id", p.ID())
            continue
        }

        // 安全措施3: 检查最新状态是否仍然是 pending
        if latestPayment.Status() != vo.PaymentStatusPending {
            uc.logger.Infow("payment status changed, skipping expiration",
                "payment_id", p.ID(),
                "status", latestPayment.Status())
            continue
        }

        // 安全措施4 (可选): 调用支付网关查询最终状态
        if uc.gateway != nil {
            gatewayStatus, err := uc.gateway.QueryPaymentStatus(ctx, latestPayment.GatewayOrderNo())
            if err == nil && gatewayStatus == "SUCCESS" {
                uc.logger.Warnw("payment gateway shows success but local status is pending",
                    "payment_id", p.ID(),
                    "gateway_order_no", latestPayment.GatewayOrderNo())
                // 触发补偿逻辑，手动处理支付成功
                uc.triggerCompensation(ctx, latestPayment)
                continue
            }
        }

        // 标记为过期
        if err := latestPayment.MarkAsExpired(); err != nil {
            uc.logger.Errorw("failed to mark payment as expired",
                "error", err,
                "payment_id", p.ID())
            continue
        }

        // 更新数据库（乐观锁保护）
        if err := uc.paymentRepo.Update(ctx, latestPayment); err != nil {
            uc.logger.Errorw("failed to update expired payment",
                "error", err,
                "payment_id", p.ID())
            continue
        }

        expiredCount++
        uc.logger.Infow("payment marked as expired",
            "payment_id", p.ID(),
            "order_no", p.OrderNo())
    }

    uc.logger.Infow("expired payments processed",
        "total", len(expiredPayments),
        "expired", expiredCount)

    return expiredCount, nil
}
```

**方案 2: 修改查询条件，排除最近过期的订单**
```go
// paymentrepository.go
func (r *PaymentRepository) GetExpiredPayments(ctx context.Context) ([]*payment.Payment, error) {
    var paymentModels []models.PaymentModel

    // 只查询过期超过5分钟的订单
    fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

    if err := r.db.WithContext(ctx).
        Where("payment_status = ? AND expired_at < ?", vo.PaymentStatusPending, fiveMinutesAgo).
        Find(&paymentModels).Error; err != nil {
        return nil, fmt.Errorf("failed to get expired payments: %w", err)
    }

    // ... 转换逻辑
}
```

#### 复现概率
**中等** - 取决于支付网关回调延迟，在高峰期更容易发生。

#### 潜在影响
- **最严重**: 用户支付成功但订单显示过期，导致退款纠纷
- 影响用户体验和平台信誉
- 增加客服成本

---

### 🔴 P0-3: 节点流量累积没有使用原子操作（严重）

**问题编号**: P0-3
**严重程度**: 🔴 严重
**问题类型**: Read-Modify-Write 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/domain/node/node.go:492-505`
- `/Users/easayliu/Documents/go/orris/internal/application/node/usecases/recordnodetraffic.go`

#### 问题描述
节点流量记录使用了 Read-Modify-Write 模式，但没有使用数据库原子操作或乐观锁保护。

**并发场景**:
```
时间线:
T1: 节点A上报流量 -> GetByID() -> trafficUsed = 100GB
T2: 节点B上报流量（同一节点） -> GetByID() -> trafficUsed = 100GB
T3: 节点A计算 -> 100GB + 10GB = 110GB -> Update()
T4: 节点B计算 -> 100GB + 5GB = 105GB -> Update()
```

**结果**: 最终 `trafficUsed = 105GB`，丢失了 A 的 10GB 流量！

#### 当前实现（有问题）
```go
// node.go:492-505
func (n *Node) RecordTraffic(upload, download uint64) error {
    if upload == 0 && download == 0 {
        return nil
    }

    // 直接在内存中累加，没有原子性保证
    n.trafficUsed += upload + download
    n.updatedAt = time.Now()
    // 注意：没有增加 version！

    if n.IsTrafficExceeded() {
    }

    return nil
}
```

**关键问题**:
1. `trafficUsed += upload + download` 是 Read-Modify-Write 操作
2. **没有增加 `version++`**，导致乐观锁机制失效！
3. 多个节点同时上报流量会导致数据丢失

#### 问题分析
这是一个经典的 **Lost Update Problem**：
1. 节点的 `RecordTraffic` 方法没有增加版本号
2. Repository Update 使用了乐观锁，但 version 未变，WHERE 条件永远匹配
3. 后执行的 Update 会覆盖先执行的结果

#### 修复方案

**方案 1: 使用数据库原子操作（推荐）**

```go
// 方案 1A: 直接使用 SQL 原子更新（无需读取）
func (r *NodeRepositoryImpl) RecordTraffic(ctx context.Context, nodeID uint, upload, download uint64) error {
    total := upload + download
    if total == 0 {
        return nil
    }

    result := r.db.WithContext(ctx).
        Model(&models.NodeModel{}).
        Where("id = ?", nodeID).
        UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", total))

    if result.Error != nil {
        return fmt.Errorf("failed to record traffic: %w", result.Error)
    }

    if result.RowsAffected == 0 {
        return errors.NewNotFoundError("node not found")
    }

    return nil
}
```

**方案 2: 修复 RecordTraffic 增加版本号 + 重试机制**

```go
// node.go
func (n *Node) RecordTraffic(upload, download uint64) error {
    if upload == 0 && download == 0 {
        return nil
    }

    n.trafficUsed += upload + download
    n.updatedAt = time.Now()
    n.version++ // 修复：增加版本号以启用乐观锁

    if n.IsTrafficExceeded() {
        // 可以发送事件通知
    }

    return nil
}

// recordnodetraffic.go - 增加重试机制
func (uc *RecordNodeTrafficUseCase) Execute(ctx context.Context, cmd RecordNodeTrafficCommand) error {
    maxRetries := 3
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        // 获取最新的节点数据
        node, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
        if err != nil {
            return err
        }

        // 记录流量
        if err := node.RecordTraffic(cmd.Upload, cmd.Download); err != nil {
            return err
        }

        // 更新节点（乐观锁保护）
        if err := uc.nodeRepo.Update(ctx, node); err != nil {
            if errors.IsConflictError(err) {
                // 乐观锁冲突，重试
                uc.logger.Warnw("optimistic lock conflict, retrying",
                    "node_id", cmd.NodeID,
                    "attempt", i+1)
                lastErr = err
                time.Sleep(time.Duration(i*100) * time.Millisecond) // 指数退避
                continue
            }
            return err
        }

        // 成功
        return nil
    }

    return fmt.Errorf("failed to record traffic after %d retries: %w", maxRetries, lastErr)
}
```

**方案 3: 使用独立的流量表 + 定期聚合**

当前系统已经有 `NodeTrafficRepository`，但实现有问题：

```go
// recordnodetraffic.go:85-114 (当前实现)
func (uc *RecordNodeTrafficUseCase) findOrCreateTraffic(
    ctx context.Context,
    cmd RecordNodeTrafficCommand,
    period time.Time,
) (*node.NodeTraffic, error) {
    // 问题：查询和创建之间有时间窗口
    existingRecords, err := uc.trafficRepo.GetTrafficStats(ctx, filter)
    if len(existingRecords) > 0 {
        return existingRecords[0], nil
    }

    // 如果两个请求同时执行到这里，都会创建新记录
    newTraffic, err := node.NewNodeTraffic(cmd.NodeID, cmd.UserID, cmd.SubscriptionID, period)
    return newTraffic, nil
}
```

**改进的方案 3**:
```go
// 使用 Upsert (ON DUPLICATE KEY UPDATE) 确保原子性
func (r *NodeTrafficRepositoryImpl) RecordTrafficAtomic(
    ctx context.Context,
    nodeID uint,
    userID *uint,
    subscriptionID *uint,
    period time.Time,
    upload, download uint64,
) error {
    total := upload + download

    // 使用 UPSERT 语句（MySQL）
    result := r.db.WithContext(ctx).Exec(`
        INSERT INTO node_traffic (node_id, user_id, subscription_id, period, upload, download, total, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
        ON DUPLICATE KEY UPDATE
            upload = upload + VALUES(upload),
            download = download + VALUES(download),
            total = total + VALUES(total),
            updated_at = NOW()
    `, nodeID, userID, subscriptionID, period, upload, download, total)

    if result.Error != nil {
        return fmt.Errorf("failed to record traffic: %w", result.Error)
    }

    return nil
}
```

#### 复现概率
**极高** - 生产环境中多个用户同时使用同一节点时，流量上报是高并发场景。

#### 潜在影响
- **数据准确性**: 流量统计不准确，影响计费和限额
- **用户体验**: 流量额度显示错误
- **财务风险**: 少计流量导致收入损失

---

### 🔴 P0-4: 节点流量重置与上报并发冲突（严重）

**问题编号**: P0-4
**严重程度**: 🔴 严重
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/domain/node/node.go:516-524`
- `/Users/easayliu/Documents/go/orris/internal/application/node/usecases/resetnodetraffic.go` (推测)

#### 问题描述
管理员重置流量时，可能正有用户在使用节点上报流量。

**并发场景**:
```
时间线:
T1: 节点当前流量 = 95GB
T2: 用户A上报 10GB -> GetByID() -> trafficUsed = 95GB
T3: 管理员重置流量 -> ResetTraffic() -> trafficUsed = 0GB -> Update()
T4: 用户A更新 -> trafficUsed = 105GB -> Update()
```

**结果**: 重置失败，流量变成 105GB 而不是 10GB！

#### 当前实现
```go
// node.go:516-524
func (n *Node) ResetTraffic() error {
    n.trafficUsed = 0
    n.trafficResetAt = time.Now()
    n.updatedAt = time.Now()
    n.version++  // 有版本号，但只能保证数据一致性，无法保证业务逻辑正确

    return nil
}
```

#### 问题分析
虽然有乐观锁保护，但会出现：
1. 如果重置先执行：后续的流量上报会因为版本不匹配而失败（流量丢失）
2. 如果流量上报先执行：重置会失败，需要重试（用户体验差）

#### 修复方案

**方案 1: 重置时使用数据库级别的原子操作**

```go
// NodeRepository 增加专门的重置方法
func (r *NodeRepositoryImpl) ResetTrafficAtomic(ctx context.Context, nodeID uint) error {
    now := time.Now()

    result := r.db.WithContext(ctx).Exec(`
        UPDATE nodes
        SET
            traffic_used = 0,
            traffic_reset_at = ?,
            updated_at = ?,
            version = version + 1
        WHERE id = ?
    `, now, now, nodeID)

    if result.Error != nil {
        return fmt.Errorf("failed to reset traffic: %w", result.Error)
    }

    if result.RowsAffected == 0 {
        return errors.NewNotFoundError("node not found")
    }

    return nil
}
```

**方案 2: 使用分布式锁**

```go
func (uc *ResetNodeTrafficUseCase) Execute(ctx context.Context, nodeID uint) error {
    lockKey := fmt.Sprintf("node:traffic:reset:%d", nodeID)

    lock, err := uc.redisClient.Obtain(ctx, lockKey, 10*time.Second, nil)
    if err != nil {
        return fmt.Errorf("failed to obtain lock: %w", err)
    }
    defer lock.Release(ctx)

    // 在锁保护下执行重置
    node, err := uc.nodeRepo.GetByID(ctx, nodeID)
    if err != nil {
        return err
    }

    if err := node.ResetTraffic(); err != nil {
        return err
    }

    // 使用原子操作更新
    return uc.nodeRepo.ResetTrafficAtomic(ctx, nodeID)
}
```

**方案 3: 使用消息队列 + 批处理**

将流量上报改为异步处理：
```go
// 1. 上报时只发送消息
func (uc *RecordNodeTrafficUseCase) Execute(ctx context.Context, cmd RecordNodeTrafficCommand) error {
    msg := &TrafficMessage{
        NodeID:   cmd.NodeID,
        Upload:   cmd.Upload,
        Download: cmd.Download,
        Time:     time.Now(),
    }

    return uc.messageQueue.Publish("traffic.updates", msg)
}

// 2. 消费者批量处理（避免并发冲突）
func (worker *TrafficWorker) ProcessBatch(messages []*TrafficMessage) error {
    // 按节点聚合流量
    aggregated := make(map[uint]uint64)
    for _, msg := range messages {
        aggregated[msg.NodeID] += msg.Upload + msg.Download
    }

    // 批量更新（原子操作）
    for nodeID, totalTraffic := range aggregated {
        uc.nodeRepo.RecordTrafficAtomic(ctx, nodeID, totalTraffic)
    }

    return nil
}
```

#### 复现概率
**低** - 手动重置流量的操作频率较低，但一旦发生影响严重。

#### 潜在影响
- 流量统计混乱
- 用户可能超额使用
- 影响系统可信度

---

### 🟡 P1-1: 订阅续费与取消并发冲突（中等）

**问题编号**: P1-1
**严重程度**: 🟡 中等
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/application/subscription/usecases/renewsubscription.go`
- `/Users/easayliu/Documents/go/orris/internal/application/subscription/usecases/cancelsubscription.go`

#### 问题描述
用户手动取消订阅的同时，系统自动续费任务正在执行。

**并发场景**:
```
时间线:
T1: 用户点击"取消订阅" -> GetByID() -> status = active
T2: 自动续费任务执行 -> GetByID() -> status = active
T3: 用户取消 -> Cancel() -> status = cancelled -> Update()
T4: 续费任务 -> Renew() -> status = active -> Update() (乐观锁失败)
```

**结果**: 乐观锁保护了数据一致性，但会导致：
1. 续费任务失败（需要重试逻辑）
2. 可能产生误导性的错误日志
3. 用户体验不佳（取消后看到续费失败通知）

#### 当前实现
```go
// renewsubscription.go:36-72
func (uc *RenewSubscriptionUseCase) Execute(ctx context.Context, cmd RenewSubscriptionCommand) error {
    sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
    if err != nil {
        return fmt.Errorf("failed to get subscription: %w", err)
    }

    plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
    if err != nil {
        return fmt.Errorf("failed to get subscription plan: %w", err)
    }

    if !plan.IsActive() {
        return fmt.Errorf("subscription plan is not active")
    }

    newEndDate := uc.calculateNewEndDate(sub.EndDate(), plan.BillingCycle())

    // 没有检查是否已取消
    if err := sub.Renew(newEndDate); err != nil {
        return fmt.Errorf("failed to renew subscription: %w", err)
    }

    if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
        // 乐观锁失败，但没有优雅处理
        return fmt.Errorf("failed to update subscription: %w", err)
    }

    return nil
}
```

#### 修复方案

**方案 1: 增加状态二次确认**

```go
func (uc *RenewSubscriptionUseCase) Execute(ctx context.Context, cmd RenewSubscriptionCommand) error {
    maxRetries := 2
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        // 每次重试都获取最新状态
        sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
        if err != nil {
            uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
            return fmt.Errorf("failed to get subscription: %w", err)
        }

        // 检查是否已取消
        if sub.Status() == vo.StatusCancelled {
            uc.logger.Infow("subscription already cancelled, skipping renewal",
                "subscription_id", cmd.SubscriptionID,
                "cancelled_at", sub.CancelledAt())
            return nil // 幂等性返回
        }

        // 检查是否可以续费
        if !sub.Status().CanRenew() {
            uc.logger.Warnw("subscription cannot be renewed",
                "subscription_id", cmd.SubscriptionID,
                "status", sub.Status())
            return fmt.Errorf("subscription cannot be renewed with status: %s", sub.Status())
        }

        plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
        if err != nil {
            return fmt.Errorf("failed to get subscription plan: %w", err)
        }

        if !plan.IsActive() {
            return fmt.Errorf("subscription plan is not active")
        }

        newEndDate := uc.calculateNewEndDate(sub.EndDate(), plan.BillingCycle())

        if err := sub.Renew(newEndDate); err != nil {
            return fmt.Errorf("failed to renew subscription: %w", err)
        }

        // 更新订阅（乐观锁保护）
        if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
            if errors.IsConflictError(err) && attempt < maxRetries {
                // 乐观锁冲突，重试
                uc.logger.Warnw("optimistic lock conflict during renewal, retrying",
                    "subscription_id", cmd.SubscriptionID,
                    "attempt", attempt+1)
                lastErr = err
                time.Sleep(time.Duration(attempt*100) * time.Millisecond)
                continue
            }
            return fmt.Errorf("failed to update subscription: %w", err)
        }

        // 成功
        uc.logger.Infow("subscription renewed successfully",
            "subscription_id", cmd.SubscriptionID,
            "new_end_date", newEndDate,
            "status", sub.Status(),
        )
        return nil
    }

    return fmt.Errorf("failed to renew subscription after retries: %w", lastErr)
}
```

**方案 2: 使用分布式锁**

```go
func (uc *RenewSubscriptionUseCase) Execute(ctx context.Context, cmd RenewSubscriptionCommand) error {
    lockKey := fmt.Sprintf("subscription:renew:%d", cmd.SubscriptionID)

    lock, err := uc.redisClient.Obtain(ctx, lockKey, 30*time.Second, nil)
    if err == redislock.ErrNotObtained {
        uc.logger.Warnw("another process is renewing this subscription",
            "subscription_id", cmd.SubscriptionID)
        return nil // 幂等性返回
    } else if err != nil {
        return fmt.Errorf("failed to obtain lock: %w", err)
    }
    defer lock.Release(ctx)

    // 在锁保护下执行续费...
}
```

#### 复现概率
**中等** - 取决于自动续费任务的执行频率和用户取消订阅的时机。

#### 潜在影响
- 乐观锁冲突导致任务失败
- 错误日志混淆运维人员
- 用户收到不必要的错误通知

---

### 🟡 P1-2: 修改套餐与订阅过期处理并发（中等）

**问题编号**: P1-2
**严重程度**: 🟡 中等
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/application/subscription/usecases/changeplan.go`
- `/Users/easayliu/Documents/go/orris/internal/domain/subscription/subscription.go:284-302`

#### 问题描述
用户升级套餐的同时，订阅正好到期。

**并发场景**:
```
时间线:
T1: 订阅 end_date = 2025-01-01 00:00:00，当前时间 = 2025-01-01 00:00:00
T2: 用户升级套餐 -> GetByID() -> status = active
T3: 过期检查任务 -> MarkAsExpired() -> status = expired -> Update()
T4: 升级套餐 -> ChangePlan() -> 检查 status != active -> 失败！
```

#### 当前实现
```go
// subscription.go:284-302
func (s *Subscription) ChangePlan(newPlanID uint) error {
    if newPlanID == 0 {
        return fmt.Errorf("new plan ID is required")
    }

    if newPlanID == s.planID {
        return nil
    }

    // 只允许 active 或 trialing 状态修改套餐
    if s.status != vo.StatusActive && s.status != vo.StatusTrialing {
        return fmt.Errorf("cannot change plan for subscription with status %s", s.status)
    }

    s.planID = newPlanID
    s.updatedAt = time.Now()
    s.version++

    return nil
}
```

#### 修复方案

**增加业务规则 + 重试机制**

```go
// changeplan.go
func (uc *ChangePlanUseCase) Execute(ctx context.Context, cmd ChangePlanCommand) error {
    maxRetries := 2
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        // 获取最新订阅状态
        sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
        if err != nil {
            return fmt.Errorf("failed to get subscription: %w", err)
        }

        // 如果订阅已过期但在宽限期内，允许修改套餐
        gracePeriod := 24 * time.Hour
        if sub.Status() == vo.StatusExpired && time.Since(sub.EndDate()) < gracePeriod {
            uc.logger.Warnw("subscription expired but within grace period, allowing plan change",
                "subscription_id", cmd.SubscriptionID,
                "expired_at", sub.EndDate())

            // 先激活订阅
            if err := sub.Activate(); err != nil {
                return fmt.Errorf("failed to reactivate subscription: %w", err)
            }
        }

        // 验证新套餐
        oldPlan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
        if err != nil {
            return fmt.Errorf("failed to get old plan: %w", err)
        }

        newPlan, err := uc.planRepo.GetByID(ctx, cmd.NewPlanID)
        if err != nil {
            return fmt.Errorf("failed to get new plan: %w", err)
        }

        if !newPlan.IsActive() {
            return fmt.Errorf("new plan is not active")
        }

        // 验证变更类型
        actualChangeType := uc.determineChangeType(oldPlan, newPlan)
        if actualChangeType != cmd.ChangeType {
            return fmt.Errorf("change type mismatch: requested %s but actual is %s", cmd.ChangeType, actualChangeType)
        }

        // 应用变更
        if err := uc.applyPlanChange(sub, cmd.NewPlanID, cmd.ChangeType); err != nil {
            return fmt.Errorf("failed to apply plan change: %w", err)
        }

        // 更新订阅（乐观锁保护）
        if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
            if errors.IsConflictError(err) && attempt < maxRetries {
                uc.logger.Warnw("optimistic lock conflict during plan change, retrying",
                    "subscription_id", cmd.SubscriptionID,
                    "attempt", attempt+1)
                lastErr = err
                time.Sleep(time.Duration(attempt*100) * time.Millisecond)
                continue
            }
            return fmt.Errorf("failed to update subscription: %w", err)
        }

        uc.logger.Infow("plan changed successfully",
            "subscription_id", cmd.SubscriptionID,
            "old_plan_id", oldPlan.ID(),
            "new_plan_id", cmd.NewPlanID,
            "change_type", cmd.ChangeType)

        return nil
    }

    return fmt.Errorf("failed to change plan after retries: %w", lastErr)
}
```

#### 复现概率
**低** - 需要精确的时间点巧合。

#### 潜在影响
- 用户无法在到期时刻升级套餐
- 影响用户体验

---

### 🟡 P1-3: 节点组删除与关联操作并发（中等）

**问题编号**: P1-3
**严重程度**: 🟡 中等
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/infrastructure/repository/nodegrouprepository.go:161-190`

#### 问题描述
管理员删除节点组的同时，有用户正在生成订阅链接（需要查询节点组）。

**并发场景**:
```
时间线:
T1: 用户请求生成订阅链接 -> GetNodesByGroupID(groupID=1)
T2: 管理员删除节点组 -> Delete(groupID=1) -> 删除关联 -> 删除主记录
T3: 用户查询返回节点列表（可能为空或部分数据）
```

#### 当前实现
```go
// nodegrouprepository.go:161-190
func (r *NodeGroupRepositoryImpl) Delete(ctx context.Context, id uint) error {
    // 使用事务删除
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1. 删除节点关联
        if err := tx.Where("node_group_id = ?", id).Delete(&models.NodeGroupNodeModel{}).Error; err != nil {
            return fmt.Errorf("failed to delete node associations: %w", err)
        }

        // 2. 删除套餐关联
        if err := tx.Where("node_group_id = ?", id).Delete(&models.NodeGroupPlanModel{}).Error; err != nil {
            return fmt.Errorf("failed to delete plan associations: %w", err)
        }

        // 3. 删除主记录
        result := tx.Delete(&models.NodeGroupModel{}, id)
        if result.Error != nil {
            return fmt.Errorf("failed to delete node group: %w", result.Error)
        }

        if result.RowsAffected == 0 {
            return errors.NewNotFoundError("node group not found")
        }

        return nil
    })
}
```

#### 问题分析
虽然使用了事务，但：
1. 读操作（GetNodesByGroupID）可能读取到部分删除的数据
2. 如果节点组关联了活跃订阅，不应允许删除

#### 修复方案

**增加业务规则检查**

```go
func (r *NodeGroupRepositoryImpl) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1. 检查是否有关联的活跃订阅套餐
        var activePlanCount int64
        err := tx.Model(&models.NodeGroupPlanModel{}).
            Joins("JOIN subscription_plans ON subscription_plans.id = node_group_plans.subscription_plan_id").
            Where("node_group_plans.node_group_id = ? AND subscription_plans.is_active = ?", id, true).
            Count(&activePlanCount).Error

        if err != nil {
            return fmt.Errorf("failed to check active plan associations: %w", err)
        }

        if activePlanCount > 0 {
            return errors.NewConflictError(
                fmt.Sprintf("cannot delete node group: %d active subscription plan(s) are associated", activePlanCount))
        }

        // 2. 检查是否有活跃订阅使用该节点组
        var activeSubscriptionCount int64
        err = tx.Table("subscriptions").
            Joins("JOIN node_group_plans ON subscriptions.plan_id = node_group_plans.subscription_plan_id").
            Where("node_group_plans.node_group_id = ? AND subscriptions.status IN ?",
                id, []string{"active", "trialing"}).
            Count(&activeSubscriptionCount).Error

        if err != nil {
            return fmt.Errorf("failed to check active subscriptions: %w", err)
        }

        if activeSubscriptionCount > 0 {
            return errors.NewConflictError(
                fmt.Sprintf("cannot delete node group: %d active subscription(s) are using it", activeSubscriptionCount))
        }

        // 3. 删除节点关联
        if err := tx.Where("node_group_id = ?", id).Delete(&models.NodeGroupNodeModel{}).Error; err != nil {
            r.logger.Errorw("failed to delete node group node associations", "id", id, "error", err)
            return fmt.Errorf("failed to delete node associations: %w", err)
        }

        // 4. 删除套餐关联
        if err := tx.Where("node_group_id = ?", id).Delete(&models.NodeGroupPlanModel{}).Error; err != nil {
            r.logger.Errorw("failed to delete node group plan associations", "id", id, "error", err)
            return fmt.Errorf("failed to delete plan associations: %w", err)
        }

        // 5. 软删除主记录
        result := tx.Delete(&models.NodeGroupModel{}, id)
        if result.Error != nil {
            r.logger.Errorw("failed to delete node group", "id", id, "error", result.Error)
            return fmt.Errorf("failed to delete node group: %w", result.Error)
        }

        if result.RowsAffected == 0 {
            return errors.NewNotFoundError("node group not found")
        }

        r.logger.Infow("node group deleted successfully", "id", id)
        return nil
    })
}
```

#### 复现概率
**低** - 删除操作频率较低。

#### 潜在影响
- 用户可能生成包含无效节点的订阅链接
- 数据一致性问题

---

### 🟡 P1-4: 创建订阅时并发创建相同订阅（中等）

**问题编号**: P1-4
**严重程度**: 🟡 中等
**问题类型**: 竞态条件
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/application/subscription/usecases/createsubscription.go`

#### 问题描述
用户快速点击多次"创建订阅"按钮，可能创建多个相同的订阅。

**并发场景**:
```
时间线:
T1: 请求A -> 检查用户订阅 -> 无活跃订阅
T2: 请求B -> 检查用户订阅 -> 无活跃订阅
T3: 请求A -> 创建订阅 -> success
T4: 请求B -> 创建订阅 -> success (重复创建！)
```

#### 当前实现
```go
// createsubscription.go:54-134
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
    // 获取套餐
    plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
    if err != nil {
        return nil, fmt.Errorf("failed to get subscription plan: %w", err)
    }

    // 没有检查是否已有pending订阅！

    // 允许多个活跃订阅（注释说明）
    // Allow multiple active subscriptions per user
    // No restriction on creating new subscriptions

    // 创建订阅...
    sub, err := subscription.NewSubscription(cmd.UserID, cmd.PlanID, startDate, endDate, cmd.AutoRenew)
    // ...
}
```

#### 问题分析
1. 没有检查用户是否已有相同套餐的 `pending` 订阅
2. 重复点击会创建多个待支付订阅

#### 修复方案

**增加重复创建检查**

```go
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
    // 1. 检查用户是否已有相同套餐的pending订阅
    existingPendingSubscriptions, err := uc.subscriptionRepo.GetByUserAndPlan(ctx, cmd.UserID, cmd.PlanID)
    if err != nil {
        return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
    }

    // 检查是否有pending或active状态的相同套餐订阅
    for _, existingSub := range existingPendingSubscriptions {
        if existingSub.Status() == vo.StatusPendingPayment {
            // 返回已存在的订阅（幂等性）
            uc.logger.Infow("subscription already exists in pending state",
                "subscription_id", existingSub.ID(),
                "user_id", cmd.UserID,
                "plan_id", cmd.PlanID)

            // 可以返回现有订阅，或者提示用户
            return &CreateSubscriptionResult{
                Subscription: existingSub,
                Token:        nil, // 需要查询已有token
            }, nil
        }

        // 如果已有active订阅同一套餐，根据业务规则决定是否允许
        if existingSub.Status() == vo.StatusActive {
            uc.logger.Warnw("user already has active subscription for this plan",
                "subscription_id", existingSub.ID(),
                "user_id", cmd.UserID,
                "plan_id", cmd.PlanID)

            // 选项1: 直接拒绝
            // return nil, errors.NewConflictError("you already have an active subscription for this plan")

            // 选项2: 允许（当前逻辑）
            // 继续创建新订阅
        }
    }

    // 获取套餐
    plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
    if err != nil {
        return nil, fmt.Errorf("failed to get subscription plan: %w", err)
    }

    if !plan.IsActive() {
        return nil, fmt.Errorf("subscription plan is not active")
    }

    // ... 其余创建逻辑
}
```

**更好的方案：使用分布式锁**

```go
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
    // 使用用户ID+套餐ID作为锁的key
    lockKey := fmt.Sprintf("subscription:create:%d:%d", cmd.UserID, cmd.PlanID)

    lock, err := uc.redisClient.Obtain(ctx, lockKey, 10*time.Second, nil)
    if err == redislock.ErrNotObtained {
        uc.logger.Warnw("another request is creating subscription for this user and plan",
            "user_id", cmd.UserID,
            "plan_id", cmd.PlanID)
        return nil, errors.NewConflictError("a subscription creation is already in progress, please wait")
    } else if err != nil {
        return nil, fmt.Errorf("failed to obtain lock: %w", err)
    }
    defer lock.Release(ctx)

    // 在锁保护下创建订阅...
}
```

#### 复现概率
**中等** - 用户快速点击或网络重试时会发生。

#### 潜在影响
- 创建重复订阅
- 用户困惑
- 数据冗余

---

### 🟡 P1-5: 定时任务在多实例部署时重复执行（中等）

**问题编号**: P1-5
**严重程度**: 🟡 中等
**问题类型**: 分布式环境下的幂等性问题
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/infrastructure/scheduler/payment_scheduler.go`

#### 问题描述
如果系统部署多个实例（高可用），每个实例都会运行定时任务，导致重复处理。

**并发场景**:
```
时间线:
实例A: 15:00 触发定时任务 -> 获取过期支付列表 [P1, P2, P3]
实例B: 15:00 触发定时任务 -> 获取过期支付列表 [P1, P2, P3]
实例A: 处理 P1, P2, P3
实例B: 处理 P1, P2, P3 (重复处理！)
```

#### 当前实现
```go
// payment_scheduler.go:30-48
func (s *PaymentScheduler) Start(ctx context.Context) {
    s.logger.Infow("starting payment scheduler", "interval", s.interval)

    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-s.stopChan:
            return
        case <-ticker.C:
            s.processExpiredPayments(ctx) // 所有实例都会执行
        }
    }
}
```

#### 问题分析
1. 没有分布式锁保护
2. 多个实例同时执行会导致重复处理
3. 虽然乐观锁能防止数据不一致，但会增加数据库负载

#### 修复方案

**方案 1: 使用分布式锁（推荐）**

```go
func (s *PaymentScheduler) processExpiredPayments(ctx context.Context) {
    lockKey := "scheduler:payment:expire"
    lockTTL := 4 * time.Minute // 略小于任务间隔

    // 尝试获取分布式锁
    lock, err := s.redisClient.Obtain(ctx, lockKey, lockTTL, &redislock.Options{
        RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), 3),
    })

    if err == redislock.ErrNotObtained {
        s.logger.Debugw("another instance is processing expired payments, skipping")
        return
    } else if err != nil {
        s.logger.Errorw("failed to obtain scheduler lock", "error", err)
        return
    }
    defer lock.Release(ctx)

    s.logger.Debugw("processing expired payments task started (lock obtained)")

    count, err := s.expirePaymentsUC.Execute(ctx)
    if err != nil {
        s.logger.Errorw("failed to process expired payments", "error", err)
        return
    }

    if count > 0 {
        s.logger.Infow("expired payments processed", "count", count)
    }
}
```

**方案 2: 使用专门的调度器（如 Leader Election）**

```go
// 使用 etcd 或 consul 实现 Leader Election
type LeaderElection struct {
    etcdClient *clientv3.Client
    sessionID  clientv3.LeaseID
    isLeader   atomic.Bool
}

func (s *PaymentScheduler) Start(ctx context.Context) {
    // 只有 Leader 节点运行定时任务
    leaderElection := NewLeaderElection(s.etcdClient)

    go leaderElection.Campaign(ctx, "scheduler/payment", func() {
        s.isLeader.Store(true)
        s.runScheduler(ctx)
    })
}

func (s *PaymentScheduler) runScheduler(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    for {
        if !s.isLeader.Load() {
            s.logger.Infow("no longer leader, stopping scheduler")
            return
        }

        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.processExpiredPayments(ctx)
        }
    }
}
```

**方案 3: 基于数据库的简单锁**

```go
// 在数据库中创建 scheduler_locks 表
CREATE TABLE scheduler_locks (
    lock_name VARCHAR(100) PRIMARY KEY,
    locked_at TIMESTAMP NOT NULL,
    locked_by VARCHAR(100) NOT NULL,
    expires_at TIMESTAMP NOT NULL
);

func (s *PaymentScheduler) processExpiredPayments(ctx context.Context) {
    instanceID := os.Getenv("INSTANCE_ID") // 或使用 hostname
    lockName := "expire_payments"
    lockDuration := 4 * time.Minute

    // 尝试获取锁
    acquired, err := s.acquireDBLock(ctx, lockName, instanceID, lockDuration)
    if err != nil {
        s.logger.Errorw("failed to acquire lock", "error", err)
        return
    }

    if !acquired {
        s.logger.Debugw("lock already held by another instance")
        return
    }

    defer s.releaseDBLock(ctx, lockName, instanceID)

    // 执行任务...
}

func (s *PaymentScheduler) acquireDBLock(ctx context.Context, lockName, instanceID string, duration time.Duration) (bool, error) {
    now := time.Now()
    expiresAt := now.Add(duration)

    // 尝试插入锁记录（如果不存在）或更新已过期的锁
    result := s.db.Exec(`
        INSERT INTO scheduler_locks (lock_name, locked_at, locked_by, expires_at)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            locked_at = IF(expires_at < ?, VALUES(locked_at), locked_at),
            locked_by = IF(expires_at < ?, VALUES(locked_by), locked_by),
            expires_at = IF(expires_at < ?, VALUES(expires_at), expires_at)
    `, lockName, now, instanceID, expiresAt, now, now, now)

    if result.Error != nil {
        return false, result.Error
    }

    // 检查是否成功获取锁
    var lockedBy string
    err := s.db.Raw("SELECT locked_by FROM scheduler_locks WHERE lock_name = ?", lockName).Scan(&lockedBy).Error
    if err != nil {
        return false, err
    }

    return lockedBy == instanceID, nil
}
```

#### 复现概率
**高** - 只要部署了多实例就会发生。

#### 潜在影响
- 数据库乐观锁冲突增加
- CPU 和数据库负载增加
- 错误日志增多

---

### 🟢 P2-1: 用户创建时邮箱唯一约束冲突处理不完善（轻微）

**问题编号**: P2-1
**严重程度**: 🟢 轻微
**问题类型**: 唯一约束冲突处理
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/infrastructure/repository/userrepositoryddd.go:32-54`

#### 问题描述
并发创建相同邮箱的用户时，数据库唯一约束会报错，但错误处理可能不够优雅。

#### 当前实现
```go
// userrepositoryddd.go:32-54
func (r *UserRepositoryDDD) Create(ctx context.Context, userEntity *user.User) error {
    model, err := r.mapper.ToModel(userEntity)
    if err != nil {
        return fmt.Errorf("failed to map user entity: %w", err)
    }

    // 直接创建，没有检查邮箱是否已存在
    if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
        r.logger.Errorw("failed to create user in database", "error", err)
        return fmt.Errorf("failed to create user: %w", err)
    }

    // ...
}
```

#### 修复方案

```go
func (r *UserRepositoryDDD) Create(ctx context.Context, userEntity *user.User) error {
    model, err := r.mapper.ToModel(userEntity)
    if err != nil {
        r.logger.Errorw("failed to map user entity to model", "error", err)
        return fmt.Errorf("failed to map user entity: %w", err)
    }

    if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
        // 检查是否是唯一约束冲突
        if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
            if strings.Contains(err.Error(), "email") {
                r.logger.Warnw("user with this email already exists", "email", model.Email)
                return errors.NewConflictError("user with this email already exists")
            }
            return errors.NewConflictError("user already exists")
        }

        r.logger.Errorw("failed to create user in database", "error", err)
        return fmt.Errorf("failed to create user: %w", err)
    }

    // Set the ID back to the entity
    if err := userEntity.SetID(model.ID); err != nil {
        r.logger.Errorw("failed to set user ID", "error", err)
        return fmt.Errorf("failed to set user ID: %w", err)
    }

    r.logger.Infow("user created successfully", "id", model.ID, "email", model.Email)
    return nil
}
```

#### 复现概率
**低** - 取决于并发注册相同邮箱的概率。

#### 潜在影响
- 返回的错误信息不够友好
- 用户体验稍差

---

### 🟢 P2-2: 节点名称唯一约束处理（轻微）

**问题编号**: P2-2
**严重程度**: 🟢 轻微
**问题类型**: 唯一约束冲突处理
**受影响文件**:
- `/Users/easayliu/Documents/go/orris/internal/infrastructure/repository/noderepository.go:34-62`

#### 问题描述
节点创建时已经正确处理了唯一约束冲突（已实现）。

#### 当前实现（已正确）
```go
if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
    if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
        if strings.Contains(err.Error(), "name") {
            return errors.NewConflictError("node with this name already exists")
        }
        if strings.Contains(err.Error(), "token_hash") {
            return errors.NewConflictError("node with this token already exists")
        }
        return errors.NewConflictError("node already exists")
    }
    // ...
}
```

#### 结论
**已正确实现** ✅，无需修改。

---

### 🟢 P2-3: 订阅 Token 生成冲突（极低概率）

**问题编号**: P2-3
**严重程度**: 🟢 轻微
**问题类型**: 理论上的冲突可能性

#### 问题描述
虽然 Token 使用 UUID 生成，理论上存在极低的冲突概率，但实际生产环境基本不会发生。

#### 建议
保持当前实现，无需特殊处理。UUID v4 的冲突概率约为 `1/2^122`，可以忽略不计。

---

## 三、优先级排序

### 🔴 P0 级（必须立即修复）

| 编号 | 问题 | 严重程度 | 复现概率 | 影响范围 |
|------|------|----------|----------|----------|
| P0-1 | 支付回调处理缺少幂等性保护 | 🔴 严重 | 高 | 支付、订阅 |
| P0-2 | 支付超时检查与回调并发冲突 | 🔴 严重 | 中等 | 支付、用户体验 |
| P0-3 | 节点流量累积没有使用原子操作 | 🔴 严重 | 极高 | 流量统计、计费 |
| P0-4 | 节点流量重置与上报并发冲突 | 🔴 严重 | 低 | 流量统计 |

### 🟡 P1 级（应该尽快修复）

| 编号 | 问题 | 严重程度 | 复现概率 | 影响范围 |
|------|------|----------|----------|----------|
| P1-1 | 订阅续费与取消并发冲突 | 🟡 中等 | 中等 | 订阅管理 |
| P1-2 | 修改套餐与订阅过期处理并发 | 🟡 中等 | 低 | 订阅管理 |
| P1-3 | 节点组删除与关联操作并发 | 🟡 中等 | 低 | 节点管理 |
| P1-4 | 创建订阅时并发创建相同订阅 | 🟡 中等 | 中等 | 订阅管理 |
| P1-5 | 定时任务在多实例部署时重复执行 | 🟡 中等 | 高 | 系统性能 |

### 🟢 P2 级（可以后续优化）

| 编号 | 问题 | 严重程度 | 复现概率 | 影响范围 |
|------|------|----------|----------|----------|
| P2-1 | 用户创建时邮箱唯一约束冲突处理 | 🟢 轻微 | 低 | 用户体验 |
| P2-2 | 节点名称唯一约束处理 | 🟢 轻微 | N/A | 已正确实现 |
| P2-3 | 订阅 Token 生成冲突 | 🟢 轻微 | 极低 | 理论风险 |

---

## 四、修复建议总结

### 短期修复建议（1-2 周内完成）

#### 1. 立即修复流量统计问题（P0-3）
**优先级**: 🔴🔴🔴 最高

流量统计直接影响计费和用户配额，必须立即修复：

```go
// 推荐方案：使用数据库原子操作
func (r *NodeRepositoryImpl) RecordTrafficAtomic(ctx context.Context, nodeID uint, upload, download uint64) error {
    total := upload + download
    if total == 0 {
        return nil
    }

    result := r.db.WithContext(ctx).
        Model(&models.NodeModel{}).
        Where("id = ?", nodeID).
        UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", total))

    return result.Error
}
```

**影响**: 保证流量统计准确性，避免财务风险。

---

#### 2. 修复支付回调幂等性问题（P0-1）
**优先级**: 🔴🔴🔴 最高

支付是核心业务，必须保证幂等性：

**步骤**:
1. 添加数据库唯一索引:
   ```sql
   ALTER TABLE payments ADD UNIQUE INDEX idx_transaction_id (transaction_id);
   ```

2. 修改回调处理逻辑（见详细方案）

3. 增加补偿机制（处理唯一约束冲突）

**影响**: 避免重复支付处理、订阅重复激活。

---

#### 3. 修复支付超时检查冲突（P0-2）
**优先级**: 🔴🔴 高

增加缓冲时间和二次确认：

```go
// 只处理过期超过 5 分钟的订单
if time.Since(p.ExpiredAt()) < 5*time.Minute {
    continue
}

// 再次检查最新状态
latestPayment, err := uc.paymentRepo.GetByID(ctx, p.ID())
if latestPayment.Status() != vo.PaymentStatusPending {
    continue
}
```

**影响**: 避免用户支付成功却被标记为过期。

---

#### 4. 实现分布式锁保护定时任务（P1-5）
**优先级**: 🟡🟡 中等

如果当前已经是多实例部署，必须添加分布式锁：

```go
lock, err := redisClient.Obtain(ctx, "scheduler:payment:expire", 4*time.Minute)
if err == redislock.ErrNotObtained {
    return // 另一个实例正在处理
}
defer lock.Release(ctx)
```

**影响**: 减少数据库负载，避免重复处理。

---

### 长期优化建议（1-3 个月内完成）

#### 1. 架构层面优化

**1.1 引入消息队列处理高并发场景**

将流量上报改为异步处理：
- 节点上报流量 → 发送消息到 Kafka/RabbitMQ
- 消费者批量聚合 → 定期写入数据库（原子操作）

**优点**:
- 降低数据库并发压力
- 提高流量上报吞吐量
- 天然解决并发问题

**缺点**:
- 增加系统复杂度
- 需要维护消息队列

---

**1.2 实现 CQRS（命令查询职责分离）**

分离读写模型：
- 写操作使用强一致性（乐观锁 + 原子操作）
- 读操作使用只读副本（提高查询性能）

---

**1.3 引入分布式事务管理器（如 Saga）**

对于跨多个聚合根的操作（如支付成功 → 激活订阅），使用 Saga 模式：
- 定义补偿操作
- 确保最终一致性
- 提高系统可靠性

---

#### 2. 监控和告警

**2.1 添加并发冲突监控**

```go
// 使用 Prometheus 记录乐观锁冲突
optimisticLockConflicts := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "optimistic_lock_conflicts_total",
        Help: "Total number of optimistic lock conflicts",
    },
    []string{"entity", "operation"},
)

// 在 Update 失败时记录
if errors.IsConflictError(err) {
    optimisticLockConflicts.WithLabelValues("payment", "update").Inc()
}
```

**2.2 设置告警规则**

```yaml
# Prometheus 告警规则
groups:
  - name: concurrency
    rules:
      - alert: HighOptimisticLockConflicts
        expr: rate(optimistic_lock_conflicts_total[5m]) > 10
        annotations:
          summary: "高并发冲突检测"
          description: "{{ $labels.entity }}.{{ $labels.operation }} 在过去 5 分钟内乐观锁冲突超过 10 次"
```

---

#### 3. 数据库优化

**3.1 添加必要的索引**

```sql
-- 支付查询优化
CREATE INDEX idx_payment_status_expired ON payments(payment_status, expired_at);

-- 订阅查询优化
CREATE INDEX idx_subscription_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscription_plan_status ON subscriptions(plan_id, status);

-- 流量查询优化
CREATE INDEX idx_traffic_node_period ON node_traffic(node_id, period);
```

**3.2 定期清理历史数据**

```go
// 删除 90 天前的流量记录
func (r *NodeTrafficRepositoryImpl) CleanupOldRecords(ctx context.Context) error {
    before := time.Now().AddDate(0, 0, -90)
    return r.DeleteOldRecords(ctx, before)
}
```

---

#### 4. 测试和验证

**4.1 编写并发测试用例**

```go
func TestPaymentCallback_Concurrency(t *testing.T) {
    // 模拟 100 个并发回调请求
    var wg sync.WaitGroup
    errors := make([]error, 100)

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            errors[idx] = uc.Execute(ctx, sameRequest)
        }(i)
    }

    wg.Wait()

    // 验证：只有一个成功，其余幂等返回
    successCount := 0
    for _, err := range errors {
        if err == nil {
            successCount++
        }
    }

    assert.Equal(t, 1, successCount, "only one callback should succeed")
}
```

**4.2 压力测试**

使用 JMeter 或 Locust 进行压力测试：
- 测试支付回调并发处理能力
- 测试流量上报吞吐量
- 测试订阅创建并发性能

---

### 最佳实践建议

#### 1. 乐观锁使用规范

✅ **正确做法**:
```go
// 1. Domain 层修改状态时增加版本号
func (s *Subscription) Activate() error {
    s.status = vo.StatusActive
    s.version++  // 必须增加版本号
    return nil
}

// 2. Repository 使用 version-1 作为 WHERE 条件
Where("id = ? AND version = ?", model.ID, model.Version-1)
```

❌ **错误做法**:
```go
// 忘记增加版本号
func (n *Node) RecordTraffic(upload, download uint64) error {
    n.trafficUsed += upload + download
    // n.version++  // 缺失！导致乐观锁失效
    return nil
}
```

---

#### 2. 幂等性设计原则

**原则 1**: 使用唯一业务标识
```go
// 使用 transaction_id 作为幂等性 key
if paymentOrder.TransactionID() != nil {
    return nil // 已处理
}
```

**原则 2**: 数据库约束 + 应用层检查
```sql
-- 数据库层面
ALTER TABLE payments ADD UNIQUE INDEX idx_transaction_id (transaction_id);
```

```go
// 应用层面
if err := db.Create(payment); err != nil {
    if isDuplicateKeyError(err) {
        return nil // 幂等性返回
    }
    return err
}
```

**原则 3**: 使用分布式锁
```go
lockKey := fmt.Sprintf("operation:%s", businessID)
lock, err := redis.Obtain(ctx, lockKey, ttl)
if err == redislock.ErrNotObtained {
    return nil // 正在处理中
}
defer lock.Release(ctx)
```

---

#### 3. 原子操作优先原则

**优先级顺序**:
1. **数据库原子操作**（最优）
   ```go
   UPDATE nodes SET traffic_used = traffic_used + ? WHERE id = ?
   ```

2. **乐观锁 + 重试**（次优）
   ```go
   for retries := 0; retries < 3; retries++ {
       node := getNode()
       node.UpdateTraffic()
       if err := repo.Update(node); !isOptimisticLockError(err) {
           break
       }
   }
   ```

3. **悲观锁/分布式锁**（最后选择）
   ```go
   lock := acquireLock()
   defer lock.Release()
   // 临界区代码
   ```

---

#### 4. 错误处理最佳实践

```go
if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
    // 区分不同类型的错误
    if errors.IsConflictError(err) {
        // 乐观锁冲突：记录日志，可能需要重试
        uc.logger.Warnw("optimistic lock conflict", "subscription_id", sub.ID())
        return errors.NewConflictError("subscription was modified by another process, please retry")
    }

    if errors.IsNotFoundError(err) {
        // 资源不存在
        return errors.NewNotFoundError("subscription not found")
    }

    // 其他错误
    uc.logger.Errorw("failed to update subscription", "error", err)
    return errors.NewInternalError("failed to update subscription")
}
```

---

## 五、总结

### 关键发现

1. ✅ **乐观锁实现正确**: 所有核心 Repository 都正确实现了乐观锁机制
2. ❌ **流量统计存在严重问题**: `RecordTraffic` 没有增加版本号，导致乐观锁失效
3. ⚠️ **幂等性不足**: 支付回调、定时任务缺少幂等性保护
4. ⚠️ **缺少分布式锁**: 多实例部署时定时任务会重复执行

### 修复优先级

1. **第一周**: 修复 P0-3（流量统计）和 P0-1（支付幂等性）
2. **第二周**: 修复 P0-2（支付超时冲突）和 P1-5（定时任务分布式锁）
3. **第三周**: 修复 P1 级别其他问题
4. **长期**: 架构优化和监控完善

### 预期收益

修复这些问题后，系统将获得：
- ✅ **数据准确性**: 流量统计、计费准确无误
- ✅ **业务可靠性**: 支付、订阅状态一致
- ✅ **系统稳定性**: 减少数据库冲突，降低错误率
- ✅ **用户体验**: 避免重复订阅、支付失败等问题
- ✅ **可扩展性**: 支持多实例部署，高可用架构

---

**报告生成时间**: 2025-11-12
**分析工具**: Claude Code
**覆盖范围**: Orris 系统核心业务流程并发安全分析
**发现问题总数**: 12 个（P0: 4 个，P1: 5 个，P2: 3 个）

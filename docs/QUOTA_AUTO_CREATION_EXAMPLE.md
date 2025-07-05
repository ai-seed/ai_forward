# 配额使用记录自动创建示例

## 📋 场景设置

**用户配置**：
- 用户ID：123
- 配额类型：requests（请求次数）
- 配额周期：month（按月）
- 配额限制：1000次/月
- 重置时间：每月1日 00:00

## 🗓️ 时间线示例

### **2024年6月1日 - 首次使用**

#### 1. 用户发起第一个API请求

```go
// 配额检查
allowed, err := quotaService.CheckQuota(ctx, 123, "requests", 1)
```

#### 2. 系统内部处理流程

```go
// 1. 计算当前周期
periodStart := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)  // 2024-06-01 00:00:00
periodEnd := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)    // 2024-07-01 00:00:00

// 2. 查询现有使用记录
usage, err := quotaUsageRepo.GetByQuotaAndPeriod(ctx, 123, 456, periodStart, periodEnd)
// 结果：没有找到记录 (err == entities.ErrUserNotFound)

// 3. 自动创建新记录
usage = &entities.QuotaUsage{
    UserID:      123,
    QuotaID:     456,
    PeriodStart: 2024-06-01 00:00:00,
    PeriodEnd:   2024-07-01 00:00:00,
    UsedValue:   0,                    // 初始使用量为0
    CreatedAt:   2024-06-01 10:30:00,
    UpdatedAt:   2024-06-01 10:30:00,
}

// 4. 检查配额是否足够
// 0 + 1 <= 1000 ✅ 允许

// 5. 消费配额
quotaService.ConsumeQuota(ctx, 123, "requests", 1)

// 6. 更新使用量
// UPDATE quota_usage SET used_value = used_value + 1 WHERE ...
// 结果：used_value = 1
```

#### 3. 数据库状态

```sql
-- quota_usage 表
| id | user_id | quota_id | period_start        | period_end          | used_value |
|----|---------|----------|---------------------|---------------------|------------|
| 1  | 123     | 456      | 2024-06-01 00:00:00 | 2024-07-01 00:00:00 | 1.0        |
```

### **2024年6月15日 - 月中使用**

#### 1. 用户发起更多请求

```go
// 配额检查
allowed, err := quotaService.CheckQuota(ctx, 123, "requests", 5)
```

#### 2. 系统内部处理

```go
// 1. 计算当前周期（同6月1日）
periodStart := 2024-06-01 00:00:00
periodEnd := 2024-07-01 00:00:00

// 2. 查询现有使用记录
usage, err := quotaUsageRepo.GetByQuotaAndPeriod(ctx, 123, 456, periodStart, periodEnd)
// 结果：找到记录，used_value = 150（假设之前已使用150次）

// 3. 检查配额
// 150 + 5 <= 1000 ✅ 允许

// 4. 消费配额
// UPDATE quota_usage SET used_value = used_value + 5 WHERE ...
// 结果：used_value = 155
```

### **2024年7月1日 - 新月份首次使用**

#### 1. 用户发起新月份的第一个请求

```go
// 配额检查
allowed, err := quotaService.CheckQuota(ctx, 123, "requests", 1)
```

#### 2. 系统内部处理

```go
// 1. 计算当前周期（新的月份）
periodStart := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)  // 2024-07-01 00:00:00
periodEnd := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)    // 2024-08-01 00:00:00

// 2. 查询现有使用记录
usage, err := quotaUsageRepo.GetByQuotaAndPeriod(ctx, 123, 456, periodStart, periodEnd)
// 结果：没有找到7月的记录 (err == entities.ErrUserNotFound)

// 3. 自动创建7月的新记录
usage = &entities.QuotaUsage{
    UserID:      123,
    QuotaID:     456,
    PeriodStart: 2024-07-01 00:00:00,
    PeriodEnd:   2024-08-01 00:00:00,
    UsedValue:   0,                    // 7月重新开始，使用量为0
    CreatedAt:   2024-07-01 09:15:00,
    UpdatedAt:   2024-07-01 09:15:00,
}

// 4. 检查配额
// 0 + 1 <= 1000 ✅ 允许

// 5. 消费配额
// 创建新记录并设置 used_value = 1
```

#### 3. 数据库状态

```sql
-- quota_usage 表（现在有两条记录）
| id | user_id | quota_id | period_start        | period_end          | used_value |
|----|---------|----------|---------------------|---------------------|------------|
| 1  | 123     | 456      | 2024-06-01 00:00:00 | 2024-07-01 00:00:00 | 1000.0     |
| 2  | 123     | 456      | 2024-07-01 00:00:00 | 2024-08-01 00:00:00 | 1.0        |
```

## 🔄 **按周配额示例**

### **配置**：
- 配额周期：week（按周）
- 配额限制：500次/周
- 周期：周一00:00开始

### **2024年第23周（6月3日-6月9日）**

```go
// 6月6日（周四）首次使用
periodStart := 2024-06-03 00:00:00  // 本周周一
periodEnd := 2024-06-10 00:00:00    // 下周周一

// 自动创建第23周的使用记录
usage = &entities.QuotaUsage{
    PeriodStart: 2024-06-03 00:00:00,
    PeriodEnd:   2024-06-10 00:00:00,
    UsedValue:   0,
}
```

### **2024年第24周（6月10日-6月16日）**

```go
// 6月10日（周一）首次使用
periodStart := 2024-06-10 00:00:00  // 新周周一
periodEnd := 2024-06-17 00:00:00    // 下周周一

// 自动创建第24周的使用记录
usage = &entities.QuotaUsage{
    PeriodStart: 2024-06-10 00:00:00,
    PeriodEnd:   2024-06-17 00:00:00,
    UsedValue:   0,
}
```

## 🎯 **关键特性**

### ✅ **自动化**
- **无需手动创建**：系统自动检测新周期并创建记录
- **懒加载**：只在需要时创建，不会预先创建未来的记录
- **零配置**：用户无需关心周期管理

### 🔒 **数据一致性**
- **唯一约束**：数据库约束确保每个周期只有一条记录
- **原子操作**：创建和更新操作是原子的
- **并发安全**：多个请求同时到达时不会创建重复记录

### 📊 **灵活性**
- **支持所有周期类型**：分钟、小时、天、周、月
- **精确的周期计算**：基于配额设置的重置时间
- **历史记录保留**：每个周期的使用记录都会保留

### 🚀 **性能优化**
- **缓存集成**：新创建的记录会立即缓存
- **批量操作**：支持批量更新使用量
- **索引优化**：数据库索引优化查询性能

## 📈 **监控和调试**

### 🔍 **查看自动创建的记录**

```sql
-- 查看用户123的所有配额使用记录
SELECT 
    id,
    quota_id,
    period_start,
    period_end,
    used_value,
    created_at
FROM quota_usage 
WHERE user_id = 123 
ORDER BY period_start;
```

### 📊 **统计信息**

```sql
-- 统计每月的配额使用情况
SELECT 
    DATE_FORMAT(period_start, '%Y-%m') as month,
    COUNT(*) as records_count,
    SUM(used_value) as total_usage
FROM quota_usage 
WHERE user_id = 123 
GROUP BY DATE_FORMAT(period_start, '%Y-%m')
ORDER BY month;
```

## 🎉 **总结**

**是的！配额使用记录会自动创建！**

- ✅ **按月配额**：每月第一次使用时自动创建当月记录
- ✅ **按周配额**：每周第一次使用时自动创建当周记录  
- ✅ **按日配额**：每天第一次使用时自动创建当日记录
- ✅ **按小时/分钟配额**：每个时间段第一次使用时自动创建记录

用户完全不需要手动管理这些记录，系统会智能地处理所有周期管理！🚀

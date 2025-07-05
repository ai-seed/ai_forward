# 配额缓存日期处理机制

## 📅 问题描述

您提出的问题非常重要：**如果6.4号开始限额，6.5号也有个限额，今天6.6号，那缓存是怎么记录的？**

这涉及到配额缓存如何处理不同日期的配额使用情况。

## 🔍 原始问题分析

### ❌ **修复前的问题**

原始实现中存在一个严重的缓存键冲突问题：

```go
// 原始的缓存键生成（有问题）
cacheKey := fmt.Sprintf("%d:%s:%s", userID, string(period), periodStart.Format("2006-01-02"))
finalKey := GetQuotaUsageCacheKey(userID, quotaType, period) // 不包含日期！
// 结果：quota_usage:123:requests:daily
```

**问题**：
- 所有日期的配额使用都使用相同的缓存键
- 6月4日、6月5日、6月6日的数据会相互覆盖
- 无法同时缓存多天的配额使用情况

## ✅ **修复后的解决方案**

### 🔧 **新的缓存键设计**

修复后的缓存键包含完整的日期信息：

```go
// 新的缓存键生成（已修复）
periodKey := s.generatePeriodKey(period, periodStart)
cacheKey := fmt.Sprintf("quota_usage:user:%d:quota:%d:period:%s", userID, quotaID, periodKey)
```

### 📊 **具体示例**

假设用户ID为123，配额ID为456，配额类型为"requests"，周期为"daily"：

#### **6月4日（2024-06-04）**
- **periodStart**: `2024-06-04 00:00:00`
- **periodKey**: `day:2024-06-04`
- **缓存键**: `quota_usage:user:123:quota:456:period:day:2024-06-04`
- **存储内容**: 6月4日的配额使用情况

#### **6月5日（2024-06-05）**
- **periodStart**: `2024-06-05 00:00:00`
- **periodKey**: `day:2024-06-05`
- **缓存键**: `quota_usage:user:123:quota:456:period:day:2024-06-05`
- **存储内容**: 6月5日的配额使用情况

#### **6月6日（2024-06-06）**
- **periodStart**: `2024-06-06 00:00:00`
- **periodKey**: `day:2024-06-06`
- **缓存键**: `quota_usage:user:123:quota:456:period:day:2024-06-06`
- **存储内容**: 6月6日的配额使用情况

## 🏗️ **周期键生成规则**

### `generatePeriodKey` 方法实现

```go
func (s *quotaServiceImpl) generatePeriodKey(period entities.QuotaPeriod, periodStart time.Time) string {
    switch period {
    case entities.QuotaPeriodMinute:
        // 格式：minute:2024-06-06:14:30
        return fmt.Sprintf("minute:%s", periodStart.Format("2006-01-02:15:04"))
    case entities.QuotaPeriodHour:
        // 格式：hour:2024-06-06:14
        return fmt.Sprintf("hour:%s", periodStart.Format("2006-01-02:15"))
    case entities.QuotaPeriodDay:
        // 格式：day:2024-06-06
        return fmt.Sprintf("day:%s", periodStart.Format("2006-01-02"))
    case entities.QuotaPeriodMonth:
        // 格式：month:2024-06
        return fmt.Sprintf("month:%s", periodStart.Format("2006-01"))
    default:
        // 默认为小时
        return fmt.Sprintf("hour:%s", periodStart.Format("2006-01-02:15"))
    }
}
```

### 📋 **不同周期的缓存键示例**

#### **分钟级配额**
- **缓存键**: `quota_usage:user:123:quota:456:period:minute:2024-06-06:14:30`
- **含义**: 2024年6月6日14:30这一分钟的配额使用

#### **小时级配额**
- **缓存键**: `quota_usage:user:123:quota:456:period:hour:2024-06-06:14`
- **含义**: 2024年6月6日14点这一小时的配额使用

#### **日级配额**
- **缓存键**: `quota_usage:user:123:quota:456:period:day:2024-06-06`
- **含义**: 2024年6月6日这一天的配额使用

#### **月级配额**
- **缓存键**: `quota_usage:user:123:quota:456:period:month:2024-06`
- **含义**: 2024年6月这一个月的配额使用

## 🔄 **缓存生命周期管理**

### ⏰ **TTL策略**

不同周期的配额使用缓存采用不同的TTL：

```go
ttl := time.Duration(2) * time.Minute // 配额使用情况缓存2分钟
```

**为什么是2分钟？**
- **实时性要求**：配额使用需要相对实时的数据
- **性能平衡**：避免过于频繁的数据库查询
- **数据一致性**：短TTL确保数据不会过时太久

### 🗑️ **缓存失效策略**

#### **主动失效**
当用户消费配额后，会主动失效相关缓存：

```go
// 失效用户配额使用情况缓存（使用模式匹配）
pattern := fmt.Sprintf("quota_usage:user:%d:*", userID)
invalidationService.BatchInvalidate(ctx, []InvalidationOperation{
    NewPatternInvalidation(pattern),
})
```

#### **模式匹配失效**
- **模式**: `quota_usage:user:123:*`
- **效果**: 删除用户123的所有配额使用缓存
- **包含**: 所有日期、所有配额、所有周期的缓存

## 📊 **实际运行示例**

### 🎯 **场景：用户在不同日期的配额使用**

假设用户123有一个日配额限制100次请求：

#### **6月4日**
1. **首次查询**：缓存未命中，查询数据库，已使用50次
2. **缓存存储**：`quota_usage:user:123:quota:456:period:day:2024-06-04` → 使用50次
3. **后续查询**：直接从缓存获取，使用50次

#### **6月5日**
1. **首次查询**：缓存未命中（新的日期），查询数据库，已使用30次
2. **缓存存储**：`quota_usage:user:123:quota:456:period:day:2024-06-05` → 使用30次
3. **后续查询**：直接从缓存获取，使用30次

#### **6月6日（今天）**
1. **首次查询**：缓存未命中（新的日期），查询数据库，已使用0次
2. **缓存存储**：`quota_usage:user:123:quota:456:period:day:2024-06-06` → 使用0次
3. **用户发起请求**：消费1次配额
4. **缓存失效**：删除 `quota_usage:user:123:quota:456:period:day:2024-06-06`
5. **下次查询**：缓存未命中，查询数据库，已使用1次

### 🔍 **Redis中的实际存储**

在Redis中，同时存在多个日期的缓存：

```
quota_usage:user:123:quota:456:period:day:2024-06-04 → {used: 50, limit: 100, ...}
quota_usage:user:123:quota:456:period:day:2024-06-05 → {used: 30, limit: 100, ...}
quota_usage:user:123:quota:456:period:day:2024-06-06 → {used: 1, limit: 100, ...}
```

## 🎯 **优势和特点**

### ✅ **解决的问题**

1. **数据隔离**：不同日期的配额使用数据完全隔离
2. **历史保留**：可以同时缓存多天的历史数据
3. **精确查询**：可以精确查询任意日期的配额使用情况
4. **无冲突**：不同日期的缓存键完全不同，不会相互覆盖

### 🚀 **性能优势**

1. **减少数据库查询**：相同日期的重复查询直接从缓存获取
2. **支持历史查询**：历史日期的数据也可以从缓存获取
3. **智能失效**：只有当前日期的缓存会在配额消费后失效

### 🔒 **数据一致性**

1. **短期TTL**：2分钟的TTL确保数据不会过时太久
2. **主动失效**：配额消费后立即失效当前缓存
3. **模式匹配**：可以批量清理用户的所有配额缓存

## 📈 **监控和调试**

### 🔍 **缓存键查看**

可以通过Redis CLI查看实际的缓存键：

```bash
# 查看用户123的所有配额使用缓存
redis-cli KEYS "quota_usage:user:123:*"

# 查看特定日期的缓存
redis-cli KEYS "quota_usage:user:123:*:day:2024-06-06"
```

### 📊 **日志记录**

系统会记录详细的缓存操作日志：

```
DEBUG: Quota usage cached successfully user_id=123 quota_id=456 cache_key=quota_usage:user:123:quota:456:period:day:2024-06-06 period=daily
```

这样您就可以清楚地看到每个日期的配额使用情况是如何被独立缓存和管理的！

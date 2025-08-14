# 支付系统SQL脚本说明

## 📋 脚本列表

### 1. 表结构创建
- **`create_payment_methods_tables.sql`** - 完整的表结构创建脚本
  - 包含支付服务商表和支付方式表
  - 无外键约束，只有逻辑关联
  - 包含索引和注释
  - 包含基础初始数据

### 2. 数据初始化
- **`init_payment_data_only.sql`** - 只插入初始数据（推荐）
  - 适用于GORM自动迁移的情况
  - 只包含数据插入，不创建表结构
  
- **`init_payment_methods_test_data.sql`** - 完整测试数据
  - 包含更多测试用的支付方式和服务商
  - 适用于开发和测试环境

### 3. 数据维护
- **`check_payment_data_integrity.sql`** - 数据完整性检查
  - 检查无效的服务商引用
  - 检查状态不一致的情况
  - 检查重复数据
  - 验证配置合理性

- **`fix_payment_data_integrity.sql`** - 数据修复脚本
  - 自动修复常见的数据不一致问题
  - 禁用无效关联的支付方式
  - 修复不合理的配置

### 4. API测试
- **`test_payment_methods_api.sh`** - API功能测试脚本
  - 测试支付方式列表API
  - 测试充值选项API
  - 包含不同参数的测试用例

## 🚀 使用方式

### 方式一：GORM自动迁移（推荐）

```bash
# 1. 启动应用，GORM会自动创建表结构
go run ./cmd/server

# 2. 插入初始数据
psql -d your_database -f scripts/init_payment_data_only.sql

# 3. 验证数据
psql -d your_database -f scripts/check_payment_data_integrity.sql
```

### 方式二：手动创建表结构

```bash
# 1. 创建表结构和基础数据
psql -d your_database -f scripts/create_payment_methods_tables.sql

# 2. 添加更多测试数据（可选）
psql -d your_database -f scripts/init_payment_methods_test_data.sql

# 3. 验证数据完整性
psql -d your_database -f scripts/check_payment_data_integrity.sql
```

## 🔧 维护操作

### 定期检查数据完整性
```bash
# 检查数据是否有问题
psql -d your_database -f scripts/check_payment_data_integrity.sql

# 如果发现问题，运行修复脚本
psql -d your_database -f scripts/fix_payment_data_integrity.sql
```

### API功能测试
```bash
# 确保服务器运行在 localhost:8080
bash scripts/test_payment_methods_api.sh
```

## 📊 数据结构说明

### 支付服务商表 (payment_providers)
- 存储第三方支付服务商信息
- 支持多种类型：gateway、direct、bank
- 无外键约束设计

### 支付方式表 (payment_methods)
- 存储前端展示的支付方式
- 通过 provider_id 逻辑关联服务商
- 包含费率、限额等配置信息

### 关键特点
- **无外键约束** - 提高性能和灵活性
- **逻辑关联** - 通过应用代码保证数据一致性
- **自动索引** - GORM和脚本会创建必要的索引
- **数据验证** - 应用层和SQL脚本双重保障

## ⚠️ 注意事项

1. **数据一致性** - 定期运行完整性检查脚本
2. **备份重要** - 修改数据前请备份数据库
3. **测试环境** - 建议先在测试环境验证脚本
4. **权限控制** - 确保数据库用户有足够权限执行脚本

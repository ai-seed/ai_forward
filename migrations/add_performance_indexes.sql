-- 性能优化索引 - AI API Gateway
-- 执行前请确保在正确的数据库上运行

-- 1. 使用日志表索引（最重要的性能优化）
-- 按用户ID和创建时间查询使用日志（常用于统计和分页）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_user_id_created 
ON usage_logs(user_id, created_at DESC);

-- 按API密钥ID和创建时间查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_api_key_id_created 
ON usage_logs(api_key_id, created_at DESC);

-- 按模型和创建时间查询（用于模型使用统计）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_model_created 
ON usage_logs(model, created_at DESC);

-- 按成功状态和创建时间查询（用于错误率统计）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_success_created 
ON usage_logs(success, created_at DESC);

-- 2. API密钥表索引
-- 按用户ID和状态查询API密钥（用户管理界面）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_user_id_status 
ON api_keys(user_id, status);

-- 按状态和创建时间查询（管理员界面）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_status_created 
ON api_keys(status, created_at DESC);

-- 3. 提供商表索引
-- 按状态和优先级查询提供商（负载均衡选择）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_providers_status_priority 
ON providers(status, priority DESC);

-- 按类型和状态查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_providers_type_status 
ON providers(provider_type, status);

-- 4. 模型表索引
-- 按提供商ID和状态查询模型
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_models_provider_id_status 
ON models(provider_id, status);

-- 按模型类型和状态查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_models_model_type_status 
ON models(model_type, status);

-- 按slug查询（API调用时的模型查找）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_models_slug 
ON models(slug);

-- 5. 配额表索引
-- 按用户ID和状态查询配额
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quotas_user_id_status 
ON quotas(user_id, status);

-- 按API密钥ID和状态查询配额
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quotas_api_key_id_status 
ON quotas(api_key_id, status);

-- 按过期时间查询（清理过期配额）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quotas_expires_at 
ON quotas(expires_at);

-- 6. 用户表索引
-- 按用户名查询（登录）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_username 
ON users(username);

-- 按邮箱查询（登录、重置密码）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email 
ON users(email);

-- 按状态和创建时间查询（管理员界面）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_status_created 
ON users(status, created_at DESC);

-- 7. 提供商模型支持表索引
-- 按模型slug查询支持的提供商（关键性能索引）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_provider_model_support_model_slug 
ON provider_model_support(model_slug);

-- 按提供商ID查询支持的模型
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_provider_model_support_provider_id 
ON provider_model_support(provider_id);

-- 复合索引：按模型slug和提供商状态查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_provider_model_support_model_provider_status 
ON provider_model_support(model_slug, provider_id);

-- 8. Midjourney任务表索引（如果存在）
-- 按用户ID和状态查询任务
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_midjourney_jobs_user_id_status 
ON midjourney_jobs(user_id, status);

-- 按任务ID查询（轮询任务状态）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_midjourney_jobs_task_id 
ON midjourney_jobs(task_id);

-- 按创建时间查询（清理旧任务）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_midjourney_jobs_created_at 
ON midjourney_jobs(created_at);

-- 9. 工具表索引（如果存在）
-- 按用户ID和状态查询工具
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tools_user_id_status 
ON tools(user_id, status);

-- 按工具类型查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tools_tool_type 
ON tools(tool_type);

-- 10. 复合索引优化特定查询
-- 使用日志的复合查询优化
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_user_model_success_created 
ON usage_logs(user_id, model, success, created_at DESC);

-- API密钥的复合查询优化
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_user_status_created 
ON api_keys(user_id, status, created_at DESC);

-- 11. 部分索引优化（仅索引活跃数据）
-- 只索引活跃的API密钥
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_active_user_id 
ON api_keys(user_id) WHERE status = 'active';

-- 只索引活跃的提供商
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_providers_active_priority 
ON providers(priority DESC) WHERE status = 'active';

-- 只索引活跃的模型
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_models_active_slug 
ON models(slug) WHERE status = 'active';

-- 只索引未过期的配额
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quotas_active 
ON quotas(user_id, api_key_id) WHERE status = 'active' AND (expires_at IS NULL OR expires_at > NOW());

COMMIT;

-- 索引创建完成后，可以运行以下查询检查索引使用情况：
/*
-- 检查表大小和索引大小
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation,
    most_common_vals,
    most_common_freqs
FROM pg_stats 
WHERE tablename IN ('usage_logs', 'api_keys', 'users', 'providers', 'models', 'quotas')
ORDER BY tablename, attname;

-- 检查索引使用情况
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE schemaname = 'public'
ORDER BY idx_tup_read DESC;

-- 检查表的扫描情况
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch
FROM pg_stat_user_tables 
WHERE schemaname = 'public'
ORDER BY seq_tup_read DESC;
*/
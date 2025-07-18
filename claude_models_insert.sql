-- Claude模型数据插入脚本
-- 可以直接在数据库中执行

-- 插入Claude模型数据
INSERT OR IGNORE INTO models (provider_id, name, slug, display_name, description, model_type, context_length, max_tokens, supports_streaming, supports_functions, status, created_at, updated_at)
VALUES 
    -- Claude 1.x 系列
    (2, 'claude-1-100k', 'claude-1-100k', 'Claude 1 100K', 'Anthropic Claude 1 模型，支持100K上下文', 'chat', 100000, 4096, 1, 0, 'active', datetime('now'), datetime('now')),
    (2, 'claude-2', 'claude-2', 'Claude 2', 'Anthropic Claude 2 基础模型', 'chat', 100000, 4096, 1, 0, 'active', datetime('now'), datetime('now')),
    (2, 'claude-2.1', 'claude-2.1', 'Claude 2.1', 'Anthropic Claude 2.1 改进版本', 'chat', 200000, 4096, 1, 0, 'active', datetime('now'), datetime('now')),
    (2, 'claude-instant-1.2', 'claude-instant-1.2', 'Claude Instant 1.2', 'Anthropic Claude Instant 快速响应模型', 'chat', 100000, 4096, 1, 0, 'active', datetime('now'), datetime('now')),
    
    -- Claude 3.5 系列
    (2, 'claude-3-5-haiku', 'claude-3-5-haiku', 'Claude 3.5 Haiku', 'Anthropic Claude 3.5 Haiku 快速模型', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-20241022', 'Claude 3.5 Haiku (2024-10-22)', 'Anthropic Claude 3.5 Haiku 2024年10月版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-haiku-latest', 'claude-3-5-haiku-latest', 'Claude 3.5 Haiku Latest', 'Anthropic Claude 3.5 Haiku 最新版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    
    (2, 'claude-3-5-sonnet', 'claude-3-5-sonnet', 'Claude 3.5 Sonnet', 'Anthropic Claude 3.5 Sonnet 平衡模型', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20240620-2', 'Claude 3.5 Sonnet (2024-06-20 v2)', 'Anthropic Claude 3.5 Sonnet 2024年6月版本v2', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-sonnet-20241022', 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet (2024-10-22)', 'Anthropic Claude 3.5 Sonnet 2024年10月版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-2', 'Claude 3.5 Sonnet (2024-10-22 v2)', 'Anthropic Claude 3.5 Sonnet 2024年10月版本v2', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-20241022-all', 'Claude 3.5 Sonnet (2024-10-22 All)', 'Anthropic Claude 3.5 Sonnet 2024年10月完整版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-5-sonnet-latest', 'claude-3-5-sonnet-latest', 'Claude 3.5 Sonnet Latest', 'Anthropic Claude 3.5 Sonnet 最新版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    
    -- Claude 3.7 系列
    (2, 'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219', 'Claude 3.7 Sonnet (2025-02-19)', 'Anthropic Claude 3.7 Sonnet 2025年2月版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-20250219-thinking', 'Claude 3.7 Sonnet Thinking (2025-02-19)', 'Anthropic Claude 3.7 Sonnet 思维链版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-latest', 'Claude 3.7 Sonnet Latest', 'Anthropic Claude 3.7 Sonnet 最新版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-3-7-sonnet-thinking', 'claude-3-7-sonnet-thinking', 'Claude 3.7 Sonnet Thinking', 'Anthropic Claude 3.7 Sonnet 思维链模型', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    
    -- Claude 4 系列 (Opus)
    (2, 'claude-opus-4-20250514', 'claude-opus-4-20250514', 'Claude Opus 4 (2025-05-14)', 'Anthropic Claude Opus 4 2025年5月版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-opus-4-20250514-thinking', 'claude-opus-4-20250514-thinking', 'Claude Opus 4 Thinking (2025-05-14)', 'Anthropic Claude Opus 4 思维链版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    
    -- Claude Sonnet 4 系列
    (2, 'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514', 'Claude Sonnet 4 (2025-05-14)', 'Anthropic Claude Sonnet 4 2025年5月版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now')),
    (2, 'claude-sonnet-4-20250514-thinking', 'claude-sonnet-4-20250514-thinking', 'Claude Sonnet 4 Thinking (2025-05-14)', 'Anthropic Claude Sonnet 4 思维链版本', 'chat', 200000, 8192, 1, 1, 'active', datetime('now'), datetime('now'));

-- 查询新插入的模型ID，用于定价数据插入
-- 注意：实际执行时需要根据具体的模型ID调整

-- 为了简化，我们使用一个临时表来存储模型ID映射
CREATE TEMPORARY TABLE temp_claude_models AS
SELECT id, slug FROM models WHERE provider_id = 2 AND slug IN (
    'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
    'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
    'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022', 
    'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
    'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
    'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
    'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
);

-- 插入输入定价
INSERT INTO model_pricing (model_id, pricing_type, price_per_unit, unit, currency, effective_from, effective_until, created_at)
SELECT 
    t.id,
    'input',
    CASE 
        WHEN t.slug = 'claude-1-100k' THEN 0.008
        WHEN t.slug = 'claude-2' THEN 0.008
        WHEN t.slug = 'claude-2.1' THEN 0.008
        WHEN t.slug = 'claude-instant-1.2' THEN 0.0008
        WHEN t.slug LIKE 'claude-3-5-haiku%' THEN 0.001
        WHEN t.slug LIKE 'claude-3-5-sonnet%' THEN 0.003
        WHEN t.slug LIKE 'claude-3-7-sonnet%' THEN 0.004
        WHEN t.slug LIKE 'claude-opus-4%' THEN 0.015
        WHEN t.slug LIKE 'claude-sonnet-4%' THEN 0.003
        ELSE 0.003
    END,
    'token',
    'USD',
    '2025-01-01 00:00:00',
    NULL,
    datetime('now')
FROM temp_claude_models t;

-- 插入输出定价
INSERT INTO model_pricing (model_id, pricing_type, price_per_unit, unit, currency, effective_from, effective_until, created_at)
SELECT 
    t.id,
    'output',
    CASE 
        WHEN t.slug = 'claude-1-100k' THEN 0.024
        WHEN t.slug = 'claude-2' THEN 0.024
        WHEN t.slug = 'claude-2.1' THEN 0.024
        WHEN t.slug = 'claude-instant-1.2' THEN 0.0024
        WHEN t.slug LIKE 'claude-3-5-haiku%' THEN 0.005
        WHEN t.slug LIKE 'claude-3-5-sonnet%' THEN 0.015
        WHEN t.slug LIKE 'claude-3-7-sonnet%' THEN 0.020
        WHEN t.slug LIKE 'claude-opus-4%' THEN 0.075
        WHEN t.slug LIKE 'claude-sonnet-4%' THEN 0.015
        ELSE 0.015
    END,
    'token',
    'USD',
    '2025-01-01 00:00:00',
    NULL,
    datetime('now')
FROM temp_claude_models t;

-- 清理临时表
DROP TABLE temp_claude_models;

-- 验证插入结果
SELECT 
    m.slug,
    m.display_name,
    mp_input.price_per_unit as input_price,
    mp_output.price_per_unit as output_price
FROM models m
LEFT JOIN model_pricing mp_input ON m.id = mp_input.model_id AND mp_input.pricing_type = 'input'
LEFT JOIN model_pricing mp_output ON m.id = mp_output.model_id AND mp_output.pricing_type = 'output'
WHERE m.provider_id = 2
ORDER BY m.slug;

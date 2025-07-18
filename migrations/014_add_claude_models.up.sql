-- 添加完整的Claude模型列表
-- 基于2025年1月的最新Claude模型

-- 插入Claude模型数据
INSERT OR IGNORE INTO models (provider_id, name, slug, display_name, description, model_type, context_length, max_tokens, supports_streaming, supports_functions, status, created_at, updated_at)
VALUES 
    -- Claude 1.x 系列
    (2, 'claude-1-100k', 'claude-1-100k', 'Claude 1 100K', 'Anthropic Claude 1 模型，支持100K上下文', 'chat', 100000, 4096, true, false, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-2', 'claude-2', 'Claude 2', 'Anthropic Claude 2 基础模型', 'chat', 100000, 4096, true, false, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-2.1', 'claude-2.1', 'Claude 2.1', 'Anthropic Claude 2.1 改进版本', 'chat', 200000, 4096, true, false, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-instant-1.2', 'claude-instant-1.2', 'Claude Instant 1.2', 'Anthropic Claude Instant 快速响应模型', 'chat', 100000, 4096, true, false, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    
    -- Claude 3.x 系列 (已有部分，补充缺失的)
    -- claude-3-haiku-20240307, claude-3-sonnet-20240229, claude-3-opus-20240229 已存在
    
    -- Claude 3.5 系列
    (2, 'claude-3-5-haiku', 'claude-3-5-haiku', 'Claude 3.5 Haiku', 'Anthropic Claude 3.5 Haiku 快速模型', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-20241022', 'Claude 3.5 Haiku (2024-10-22)', 'Anthropic Claude 3.5 Haiku 2024年10月版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-haiku-latest', 'claude-3-5-haiku-latest', 'Claude 3.5 Haiku Latest', 'Anthropic Claude 3.5 Haiku 最新版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    
    (2, 'claude-3-5-sonnet', 'claude-3-5-sonnet', 'Claude 3.5 Sonnet', 'Anthropic Claude 3.5 Sonnet 平衡模型', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20240620-2', 'Claude 3.5 Sonnet (2024-06-20 v2)', 'Anthropic Claude 3.5 Sonnet 2024年6月版本v2', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022', 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet (2024-10-22)', 'Anthropic Claude 3.5 Sonnet 2024年10月版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-2', 'Claude 3.5 Sonnet (2024-10-22 v2)', 'Anthropic Claude 3.5 Sonnet 2024年10月版本v2', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-20241022-all', 'Claude 3.5 Sonnet (2024-10-22 All)', 'Anthropic Claude 3.5 Sonnet 2024年10月完整版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-latest', 'claude-3-5-sonnet-latest', 'Claude 3.5 Sonnet Latest', 'Anthropic Claude 3.5 Sonnet 最新版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    
    -- Claude 3.7 系列
    (2, 'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219', 'Claude 3.7 Sonnet (2025-02-19)', 'Anthropic Claude 3.7 Sonnet 2025年2月版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-20250219-thinking', 'Claude 3.7 Sonnet Thinking (2025-02-19)', 'Anthropic Claude 3.7 Sonnet 思维链版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-latest', 'Claude 3.7 Sonnet Latest', 'Anthropic Claude 3.7 Sonnet 最新版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-thinking', 'claude-3-7-sonnet-thinking', 'Claude 3.7 Sonnet Thinking', 'Anthropic Claude 3.7 Sonnet 思维链模型', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    
    -- Claude 4 系列 (Opus)
    (2, 'claude-opus-4-20250514', 'claude-opus-4-20250514', 'Claude Opus 4 (2025-05-14)', 'Anthropic Claude Opus 4 2025年5月版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-opus-4-20250514-thinking', 'claude-opus-4-20250514-thinking', 'Claude Opus 4 Thinking (2025-05-14)', 'Anthropic Claude Opus 4 思维链版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    
    -- Claude Sonnet 4 系列
    (2, 'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514', 'Claude Sonnet 4 (2025-05-14)', 'Anthropic Claude Sonnet 4 2025年5月版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-sonnet-4-20250514-thinking', 'claude-sonnet-4-20250514-thinking', 'Claude Sonnet 4 Thinking (2025-05-14)', 'Anthropic Claude Sonnet 4 思维链版本', 'chat', 200000, 8192, true, true, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- 插入Claude模型定价数据
-- 基于2025年1月的最新定价信息
INSERT INTO model_pricing (model_id, pricing_type, price_per_unit, unit, currency, effective_from, effective_until, created_at)
SELECT
    m.id as model_id,
    'input' as pricing_type,
    CASE
        -- Claude 1.x 系列定价
        WHEN m.slug = 'claude-1-100k' THEN 0.008
        WHEN m.slug = 'claude-2' THEN 0.008
        WHEN m.slug = 'claude-2.1' THEN 0.008
        WHEN m.slug = 'claude-instant-1.2' THEN 0.0008

        -- Claude 3.5 Haiku 系列定价
        WHEN m.slug LIKE 'claude-3-5-haiku%' THEN 0.001

        -- Claude 3.5 Sonnet 系列定价
        WHEN m.slug LIKE 'claude-3-5-sonnet%' THEN 0.003

        -- Claude 3.7 Sonnet 系列定价
        WHEN m.slug LIKE 'claude-3-7-sonnet%' THEN 0.004

        -- Claude 4 Opus 系列定价
        WHEN m.slug LIKE 'claude-opus-4%' THEN 0.015

        -- Claude Sonnet 4 系列定价
        WHEN m.slug LIKE 'claude-sonnet-4%' THEN 0.003

        ELSE 0.003 -- 默认定价
    END as price_per_unit,
    'token' as unit,
    'USD' as currency,
    '2025-01-01 00:00:00' as effective_from,
    NULL as effective_until,
    CURRENT_TIMESTAMP as created_at
FROM models m
WHERE m.provider_id = 2
AND m.slug IN (
    'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
    'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
    'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022',
    'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
    'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
    'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
    'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
);

-- 插入输出定价
INSERT INTO model_pricing (model_id, pricing_type, price_per_unit, unit, currency, effective_from, effective_until, created_at)
SELECT
    m.id as model_id,
    'output' as pricing_type,
    CASE
        -- Claude 1.x 系列定价
        WHEN m.slug = 'claude-1-100k' THEN 0.024
        WHEN m.slug = 'claude-2' THEN 0.024
        WHEN m.slug = 'claude-2.1' THEN 0.024
        WHEN m.slug = 'claude-instant-1.2' THEN 0.0024

        -- Claude 3.5 Haiku 系列定价
        WHEN m.slug LIKE 'claude-3-5-haiku%' THEN 0.005

        -- Claude 3.5 Sonnet 系列定价
        WHEN m.slug LIKE 'claude-3-5-sonnet%' THEN 0.015

        -- Claude 3.7 Sonnet 系列定价
        WHEN m.slug LIKE 'claude-3-7-sonnet%' THEN 0.020

        -- Claude 4 Opus 系列定价
        WHEN m.slug LIKE 'claude-opus-4%' THEN 0.075

        -- Claude Sonnet 4 系列定价
        WHEN m.slug LIKE 'claude-sonnet-4%' THEN 0.015

        ELSE 0.015 -- 默认定价
    END as price_per_unit,
    'token' as unit,
    'USD' as currency,
    '2025-01-01 00:00:00' as effective_from,
    NULL as effective_until,
    CURRENT_TIMESTAMP as created_at
FROM models m
WHERE m.provider_id = 2
AND m.slug IN (
    'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
    'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
    'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022',
    'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
    'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
    'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
    'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
);

-- 插入提供商模型支持关系
-- 为Anthropic提供商(provider_id=2)添加对新Claude模型的支持
INSERT OR IGNORE INTO provider_model_support (provider_id, model_slug, upstream_model_name, enabled, priority, created_at, updated_at)
VALUES
    -- Claude 1.x 系列
    (2, 'claude-1-100k', 'claude-1-100k', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-2', 'claude-2', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-2.1', 'claude-2.1', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-instant-1.2', 'claude-instant-1.2', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Claude 3.5 系列
    (2, 'claude-3-5-haiku', 'claude-3-5-haiku', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-20241022', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-haiku-latest', 'claude-3-5-haiku-latest', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    (2, 'claude-3-5-sonnet', 'claude-3-5-sonnet', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20240620-2', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022', 'claude-3-5-sonnet-20241022', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-2', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-20241022-all', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-5-sonnet-latest', 'claude-3-5-sonnet-latest', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Claude 3.7 系列
    (2, 'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-20250219-thinking', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-latest', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-3-7-sonnet-thinking', 'claude-3-7-sonnet-thinking', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Claude 4 系列 (Opus)
    (2, 'claude-opus-4-20250514', 'claude-opus-4-20250514', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-opus-4-20250514-thinking', 'claude-opus-4-20250514-thinking', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Claude Sonnet 4 系列
    (2, 'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'claude-sonnet-4-20250514-thinking', 'claude-sonnet-4-20250514-thinking', true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- Midjourney 数据库配置示例
-- 用于设置 Midjourney 模型和提供商的数据库记录

-- 1. 添加 Midjourney 提供商
INSERT INTO providers (
    name, 
    slug, 
    base_url, 
    api_key_encrypted, 
    status, 
    health_status, 
    priority, 
    description,
    created_at,
    updated_at
) VALUES 
-- 302AI Midjourney 提供商
(
    '302AI Midjourney', 
    '302ai-midjourney', 
    'https://api.302.ai', 
    'your-302ai-api-key-here',  -- 请替换为实际的 API 密钥
    'active', 
    'healthy', 
    1, 
    '302AI 提供的 Midjourney 图像生成服务',
    NOW(),
    NOW()
),
-- 官方 Midjourney 提供商（如果有的话）
(
    'Midjourney Official', 
    'midjourney-official', 
    'https://api.midjourney.com', 
    'your-official-api-key-here',  -- 请替换为实际的 API 密钥
    'active', 
    'healthy', 
    2, 
    '官方 Midjourney API 服务',
    NOW(),
    NOW()
),
-- 其他第三方提供商示例
(
    'GoAPI Midjourney', 
    'goapi-midjourney', 
    'https://api.goapi.ai', 
    'your-goapi-key-here',  -- 请替换为实际的 API 密钥
    'active', 
    'healthy', 
    3, 
    'GoAPI 提供的 Midjourney 服务',
    NOW(),
    NOW()
);

-- 2. 添加 Midjourney 模型
INSERT INTO models (
    name, 
    slug, 
    display_name, 
    model_type, 
    description, 
    status, 
    max_tokens, 
    context_window,
    created_at,
    updated_at
) VALUES 
-- 默认 Midjourney 模型
(
    'midjourney-default', 
    'midjourney-default', 
    'Midjourney 默认模型', 
    'image', 
    'Midjourney 默认的图像生成模型，支持 imagine、action、blend、describe 等操作',
    'active', 
    0,  -- 图像模型不使用 token 限制
    0,  -- 图像模型不使用上下文窗口
    NOW(),
    NOW()
),
-- Midjourney V6 模型
(
    'midjourney-v6', 
    'midjourney-v6', 
    'Midjourney V6', 
    'image', 
    'Midjourney V6 版本，提供更高质量的图像生成',
    'active', 
    0, 
    0,
    NOW(),
    NOW()
),
-- Midjourney V5 模型
(
    'midjourney-v5', 
    'midjourney-v5', 
    'Midjourney V5', 
    'image', 
    'Midjourney V5 版本，经典的图像生成模型',
    'active', 
    0, 
    0,
    NOW(),
    NOW()
),
-- Midjourney Niji 模型（动漫风格）
(
    'midjourney-niji', 
    'midjourney-niji', 
    'Midjourney Niji', 
    'image', 
    'Midjourney Niji 模型，专门用于生成动漫风格的图像',
    'active', 
    0, 
    0,
    NOW(),
    NOW()
);

-- 3. 建立模型与提供商的关联关系
-- 注意：需要先获取实际的 provider_id 和 model_id

-- 获取提供商 ID（在实际使用时需要替换为真实的 ID）
-- 302AI 支持所有 Midjourney 模型
INSERT INTO provider_model_support (
    provider_id, 
    model_slug, 
    upstream_model_name, 
    enabled, 
    priority, 
    config,
    created_at,
    updated_at
) 
SELECT 
    p.id as provider_id,
    m.slug as model_slug,
    m.slug as upstream_model_name,  -- 上游模型名称，可能与本地不同
    true as enabled,
    1 as priority,
    '{"bot_type": "MID_JOURNEY"}' as config,  -- JSON 配置
    NOW() as created_at,
    NOW() as updated_at
FROM providers p, models m 
WHERE p.slug = '302ai-midjourney' 
  AND m.model_type = 'image' 
  AND m.slug LIKE 'midjourney%';

-- 官方 Midjourney 支持部分模型
INSERT INTO provider_model_support (
    provider_id, 
    model_slug, 
    upstream_model_name, 
    enabled, 
    priority, 
    config,
    created_at,
    updated_at
) 
SELECT 
    p.id as provider_id,
    m.slug as model_slug,
    CASE 
        WHEN m.slug = 'midjourney-default' THEN 'midjourney'
        WHEN m.slug = 'midjourney-v6' THEN 'midjourney-v6'
        WHEN m.slug = 'midjourney-v5' THEN 'midjourney-v5'
        WHEN m.slug = 'midjourney-niji' THEN 'niji'
        ELSE m.slug
    END as upstream_model_name,
    true as enabled,
    2 as priority,  -- 优先级低于 302AI
    '{"bot_type": "MID_JOURNEY"}' as config,
    NOW() as created_at,
    NOW() as updated_at
FROM providers p, models m 
WHERE p.slug = 'midjourney-official' 
  AND m.model_type = 'image' 
  AND m.slug IN ('midjourney-default', 'midjourney-v6', 'midjourney-v5', 'midjourney-niji');

-- GoAPI 支持基础模型
INSERT INTO provider_model_support (
    provider_id, 
    model_slug, 
    upstream_model_name, 
    enabled, 
    priority, 
    config,
    created_at,
    updated_at
) 
SELECT 
    p.id as provider_id,
    m.slug as model_slug,
    m.slug as upstream_model_name,
    true as enabled,
    3 as priority,  -- 优先级最低
    '{"bot_type": "MID_JOURNEY"}' as config,
    NOW() as created_at,
    NOW() as updated_at
FROM providers p, models m 
WHERE p.slug = 'goapi-midjourney' 
  AND m.model_type = 'image' 
  AND m.slug IN ('midjourney-default', 'midjourney-v5');

-- 4. 添加模型定价信息（可选）
INSERT INTO model_pricing (
    model_id, 
    input_price, 
    output_price, 
    unit, 
    currency, 
    effective_date,
    created_at,
    updated_at
) 
SELECT 
    m.id as model_id,
    0.0 as input_price,     -- Midjourney 通常按图像计费，不按 token
    0.02 as output_price,   -- 每张图像 $0.02（示例价格）
    'image' as unit,        -- 计费单位：图像
    'USD' as currency,
    NOW() as effective_date,
    NOW() as created_at,
    NOW() as updated_at
FROM models m 
WHERE m.model_type = 'image' AND m.slug LIKE 'midjourney%';

-- 5. 验证数据插入
-- 查看插入的提供商
SELECT id, name, slug, base_url, status, health_status, priority 
FROM providers 
WHERE slug LIKE '%midjourney%' 
ORDER BY priority;

-- 查看插入的模型
SELECT id, name, slug, model_type, status 
FROM models 
WHERE model_type = 'image' 
ORDER BY name;

-- 查看模型与提供商的关联
SELECT 
    pms.id,
    p.name as provider_name,
    p.slug as provider_slug,
    pms.model_slug,
    pms.upstream_model_name,
    pms.enabled,
    pms.priority,
    pms.config
FROM provider_model_support pms
JOIN providers p ON pms.provider_id = p.id
WHERE pms.model_slug LIKE 'midjourney%'
ORDER BY pms.model_slug, pms.priority;

-- 查看定价信息
SELECT 
    m.name as model_name,
    m.slug as model_slug,
    mp.input_price,
    mp.output_price,
    mp.unit,
    mp.currency
FROM model_pricing mp
JOIN models m ON mp.model_id = m.id
WHERE m.model_type = 'image'
ORDER BY m.name;

-- 6. 测试查询（验证配置是否正确）
-- 查询支持 midjourney-default 模型的提供商
SELECT 
    p.id,
    p.name,
    p.slug,
    p.base_url,
    p.status,
    p.health_status,
    pms.upstream_model_name,
    pms.priority,
    pms.enabled
FROM providers p
JOIN provider_model_support pms ON p.id = pms.provider_id
WHERE pms.model_slug = 'midjourney-default'
  AND p.status = 'active'
  AND p.health_status = 'healthy'
  AND pms.enabled = true
ORDER BY pms.priority;

-- 注意事项：
-- 1. 请将 'your-xxx-api-key-here' 替换为实际的 API 密钥
-- 2. API 密钥应该加密存储，这里为了演示使用明文
-- 3. 根据实际的提供商 API 调整 base_url
-- 4. 根据实际定价调整 model_pricing 中的价格
-- 5. 可以根据需要添加更多的提供商和模型

-- 回滚Claude模型添加操作

-- 删除提供商模型支持关系
DELETE FROM provider_model_support
WHERE provider_id = 2
AND model_slug IN (
    'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
    'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
    'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022',
    'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
    'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
    'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
    'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
);

-- 删除新增的Claude模型定价数据
DELETE FROM model_pricing 
WHERE model_id IN (
    SELECT id FROM models 
    WHERE provider_id = 2 
    AND slug IN (
        'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
        'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
        'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022', 
        'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
        'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
        'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
        'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
    )
);

-- 删除新增的Claude模型数据
DELETE FROM models 
WHERE provider_id = 2 
AND slug IN (
    'claude-1-100k', 'claude-2', 'claude-2.1', 'claude-instant-1.2',
    'claude-3-5-haiku', 'claude-3-5-haiku-20241022', 'claude-3-5-haiku-latest',
    'claude-3-5-sonnet', 'claude-3-5-sonnet-20240620-2', 'claude-3-5-sonnet-20241022', 
    'claude-3-5-sonnet-20241022-2', 'claude-3-5-sonnet-20241022-all', 'claude-3-5-sonnet-latest',
    'claude-3-7-sonnet-20250219', 'claude-3-7-sonnet-20250219-thinking', 'claude-3-7-sonnet-latest', 'claude-3-7-sonnet-thinking',
    'claude-opus-4-20250514', 'claude-opus-4-20250514-thinking',
    'claude-sonnet-4-20250514', 'claude-sonnet-4-20250514-thinking'
);

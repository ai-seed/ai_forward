import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import Grid from '@mui/system/Grid';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import CardContent from '@mui/material/CardContent';
import CardActions from '@mui/material/CardActions';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';

import { Iconify } from 'src/components/iconify';
import { api } from 'src/services/api';

// ----------------------------------------------------------------------

// 厂商信息类型
interface ProviderInfo {
  id: number;
  name: string;
  display_name: string;
  color: string;
  sort_order: number;
}

// 数据库模型数据类型
interface DatabaseModel {
  id: number;
  name: string;
  slug: string;
  display_name?: string;
  description?: string;
  model_type: 'chat' | 'completion' | 'embedding' | 'image' | 'audio';
  provider: ProviderInfo;
  context_length?: number;
  max_tokens?: number;
  supports_streaming: boolean;
  supports_functions: boolean;
  status: 'active' | 'deprecated' | 'disabled';
  created_at: string;
  updated_at: string;
}

// 前端展示用的模型数据类型
interface Model {
  id: string;
  name: string;
  provider: string;
  description: string;
  category: string;
  type: 'text' | 'image' | 'audio' | 'video' | 'multimodal';
  pricing: {
    input: number;  // per 1K tokens
    output: number; // per 1K tokens
    unit: string;
  };
  capabilities: string[];
  maxTokens: number;
  status: 'available' | 'beta' | 'deprecated';
  icon: string;
  color: string;
}

// 将数据库模型转换为前端展示格式
const convertDatabaseModelToDisplayModel = (dbModel: DatabaseModel): Model => {
  // 这些函数不再需要，因为我们直接使用数据库中的厂商信息

  // 根据模型类型转换为前端类型
  const getDisplayType = (modelType: string): 'text' | 'image' | 'audio' | 'video' | 'multimodal' => {
    switch (modelType) {
      case 'chat':
      case 'completion':
        return 'text';
      case 'image':
        return 'image';
      case 'audio':
        return 'audio';
      case 'embedding':
        return 'text';
      default:
        return 'text';
    }
  };

  // 根据模型类型获取图标
  const getIcon = (modelType: string): string => {
    switch (modelType) {
      case 'image':
        return 'solar:gallery-bold-duotone';
      case 'audio':
        return 'solar:microphone-bold-duotone';
      default:
        return 'solar:cpu-bolt-bold-duotone';
    }
  };

  // getColor函数不再需要，直接使用厂商的品牌颜色

  // 根据模型类型获取能力
  const getCapabilities = (modelType: string, supportsFunctions: boolean): string[] => {
    const capabilities = [];
    if (modelType === 'chat' || modelType === 'completion') {
      capabilities.push('Text Generation');
      if (supportsFunctions) capabilities.push('Function Calling');
      capabilities.push('Reasoning');
    }
    if (modelType === 'image') {
      capabilities.push('Image Generation');
      capabilities.push('Text to Image');
    }
    if (modelType === 'audio') {
      capabilities.push('Speech Processing');
    }
    return capabilities;
  };

  const providerDisplayName = dbModel.provider.display_name;
  const displayType = getDisplayType(dbModel.model_type);

  return {
    id: dbModel.slug,
    name: dbModel.display_name || dbModel.name,
    provider: providerDisplayName,
    description: dbModel.description || `${providerDisplayName} ${dbModel.name} model`,
    category: providerDisplayName, // 按厂商分类
    type: displayType,
    pricing: {
      input: 0.003, // 默认价格，后续可以从定价表获取
      output: 0.015,
      unit: displayType === 'image' ? 'image' : '1K tokens'
    },
    capabilities: getCapabilities(dbModel.model_type, dbModel.supports_functions),
    maxTokens: dbModel.max_tokens || dbModel.context_length || 0,
    status: dbModel.status === 'active' ? 'available' : 'deprecated',
    icon: getIcon(dbModel.model_type),
    color: dbModel.provider.color // 使用厂商的品牌颜色
  };
};

// 模型类型常量（保持不变）
const TYPES = ['All', 'text', 'image', 'audio', 'video', 'multimodal'];

// ----------------------------------------------------------------------

export function ModelsView() {
  const { t } = useTranslation();
  const [selectedCategory, setSelectedCategory] = useState('All');
  const [selectedType, setSelectedType] = useState('All');
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 获取模型数据
  const fetchModels = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // 从tools/models获取数据（公开接口）
      const response = await api.noAuth.get('/tools/models');

      if (response.success && response.data) {
        // tools/models返回的是直接的数组格式
        const dbModels: DatabaseModel[] = response.data.map((item: any) => ({
          id: item.id || 0,
          name: item.name || '',
          slug: item.slug || '',
          display_name: item.display_name || item.name,
          description: item.description || '',
          model_type: item.model_type || 'chat',
          provider: item.provider || {
            id: 0,
            name: 'unknown',
            display_name: 'Unknown',
            color: '#6B7280',
            sort_order: 999
          },
          context_length: item.context_length,
          max_tokens: item.max_tokens,
          supports_streaming: item.supports_streaming || false,
          supports_functions: item.supports_functions || false,
          status: item.status || 'active',
          created_at: item.created_at || new Date().toISOString(),
          updated_at: item.updated_at || new Date().toISOString()
        }));

        const convertedModels = dbModels.map(convertDatabaseModelToDisplayModel);
        setModels(convertedModels);
      } else {
        throw new Error('Failed to fetch models');
      }
    } catch (err) {
      console.error('Error fetching models:', err);
      setError('Failed to load models');
      // 设置一些默认模型作为后备
      setModels([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  const handleCategoryChange = useCallback((category: string) => {
    setSelectedCategory(category);
  }, []);

  const handleTypeChange = useCallback((type: string) => {
    setSelectedType(type);
  }, []);

  // 动态生成厂商分类列表（按排序顺序）
  const categories = useMemo(() => {
    const providerCategories = Array.from(new Set(models.map(model => model.category)))
      .sort(); // 按字母顺序排序，后续可以按厂商的sort_order排序
    return ['All', ...providerCategories];
  }, [models]);

  const filteredModels = models.filter(model => {
    const categoryMatch = selectedCategory === 'All' || model.category === selectedCategory;
    const typeMatch = selectedType === 'All' || model.type === selectedType;
    return categoryMatch && typeMatch;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'available':
        return 'success';
      case 'beta':
        return 'warning';
      case 'deprecated':
        return 'error';
      default:
        return 'default';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'text':
        return 'solar:pen-bold';
      case 'image':
        return 'solar:eye-bold';
      case 'audio':
        return 'solar:share-bold';
      case 'video':
        return 'solar:cart-3-bold';
      case 'multimodal':
        return 'solar:restart-bold';
      default:
        return 'solar:pen-bold';
    }
  };

  const formatPricing = (model: Model) => {
    if (model.type === 'text' || model.type === 'multimodal') {
      return `$${model.pricing.input}/$${model.pricing.output} per ${model.pricing.unit}`;
    }
    return `$${model.pricing.input} per ${model.pricing.unit}`;
  };

  // 获取厂商标签颜色
  const getProviderColor = (provider: string): 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info' => {
    const colorMap: Record<string, 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info'> = {
      'OpenAI': 'success',
      'Anthropic': 'secondary',
      'Google': 'primary',
      'Meta': 'info',
      'Midjourney': 'warning',
      'Stability AI': 'error',
      'Mistral AI': 'secondary',
      'Cohere': 'primary',
      'Microsoft': 'info'
    };
    return colorMap[provider] || 'primary';
  };

  // 如果正在加载，显示加载状态
  if (loading) {
    return (
      <Box sx={{ p: 3, display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
        <CircularProgress />
      </Box>
    );
  }

  // 如果有错误，显示错误信息
  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
        <Button variant="contained" onClick={fetchModels}>
          {t('common.retry', 'Retry')}
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ mb: 1 }}>
          {t('models.title')}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          {t('models.description')}
        </Typography>
      </Box>

      {/* 筛选器 */}
      <Box sx={{ mb: 4 }}>
        <Box sx={{ mb: 3 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>
            {t('models.categories')}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            {categories.map((category) => (
              <Chip
                key={category}
                label={category === 'All' ? t('models.all_providers', 'All Providers') : category}
                onClick={() => handleCategoryChange(category)}
                variant={selectedCategory === category ? 'filled' : 'outlined'}
                color={selectedCategory === category ? 'primary' : 'default'}
                sx={{ cursor: 'pointer' }}
              />
            ))}
          </Box>
        </Box>

        <Box>
          <Typography variant="h6" sx={{ mb: 2 }}>
            {t('models.types')}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            {TYPES.map((type) => (
              <Chip
                key={type}
                label={t(`models.type_${type.toLowerCase()}`)}
                onClick={() => handleTypeChange(type)}
                variant={selectedType === type ? 'filled' : 'outlined'}
                color={selectedType === type ? 'secondary' : 'default'}
                sx={{ cursor: 'pointer' }}
                icon={<Iconify icon={getTypeIcon(type)} />}
              />
            ))}
          </Box>
        </Box>
      </Box>

      {/* 模型网格 */}
      <Grid container spacing={3}>
        {filteredModels.map((model) => (
          <Grid key={model.id} size={{ xs: 12, md: 6, lg: 4 }}>
            <Card 
              sx={{ 
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                transition: 'all 0.3s ease',
                '&:hover': {
                  transform: 'translateY(-4px)',
                  boxShadow: (theme) => theme.shadows[8],
                }
              }}
            >
              <CardContent sx={{ flexGrow: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <Box
                    sx={{
                      width: 48,
                      height: 48,
                      borderRadius: 2,
                      bgcolor: model.color,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      mr: 2,
                    }}
                  >
                    <Iconify
                      icon={getTypeIcon(model.type)}
                      sx={{ width: 24, height: 24, color: 'white' }}
                    />
                  </Box>
                  <Box sx={{ flexGrow: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                      <Typography variant="h6">
                        {model.name}
                      </Typography>
                      <Chip
                        label={model.provider}
                        size="small"
                        color={getProviderColor(model.provider)}
                        variant="filled"
                        sx={{
                          height: 20,
                          fontSize: '0.7rem',
                          fontWeight: 600
                        }}
                      />
                    </Box>
                    <Typography variant="caption" color="text.secondary">
                      {model.type} • {model.category}
                    </Typography>
                  </Box>
                  <Chip
                    label={model.status}
                    size="small"
                    color={getStatusColor(model.status) as any}
                    variant="outlined"
                  />
                </Box>

                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {model.description}
                </Typography>

                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                    {t('models.pricing')}:
                  </Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600, color: 'primary.main' }}>
                    {formatPricing(model)}
                  </Typography>
                </Box>

                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                    {t('models.capabilities')}:
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {model.capabilities.map((capability, index) => (
                      <Chip
                        key={index}
                        label={capability}
                        size="small"
                        variant="outlined"
                        sx={{ fontSize: '0.75rem' }}
                      />
                    ))}
                  </Box>
                </Box>

                {model.maxTokens > 0 && (
                  <Box>
                    <Typography variant="caption" color="text.secondary">
                      {t('models.max_tokens')}: {model.maxTokens.toLocaleString()}
                    </Typography>
                  </Box>
                )}
              </CardContent>

              <CardActions sx={{ p: 2, pt: 0 }}>
                <Button
                  fullWidth
                  variant="outlined"
                  startIcon={<Iconify icon="solar:eye-bold" />}
                >
                  {t('models.view_details')}
                </Button>
              </CardActions>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* 空状态 */}
      {filteredModels.length === 0 && (
        <Box sx={{ textAlign: 'center', py: 8 }}>
          <Iconify 
            icon="eva:search-fill"
            sx={{ width: 64, height: 64, color: 'text.disabled', mb: 2 }} 
          />
          <Typography variant="h6" color="text.secondary">
            {t('models.no_models_found')}
          </Typography>
          <Typography variant="body2" color="text.disabled">
            {t('models.try_different_filter')}
          </Typography>
        </Box>
      )}
    </Box>
  );
}

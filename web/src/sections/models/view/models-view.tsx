import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Paper from '@mui/material/Paper';

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
  pricing?: {
    input: number;
    output: number;
    unit: string;
  };
  rate_multiplier?: number;
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
  rateMultiplier?: number; // 倍率 (浮点数)
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



  const result = {
    id: dbModel.slug,
    name: dbModel.display_name || dbModel.name,
    provider: providerDisplayName,
    description: dbModel.description || `${providerDisplayName} ${dbModel.name} model`,
    category: providerDisplayName, // 按厂商分类
    type: displayType,
    pricing: dbModel.pricing || {
      input: 0.003, // 默认价格
      output: 0.015,
      unit: displayType === 'image' ? 'image' : '1K tokens'
    },
    capabilities: getCapabilities(dbModel.model_type, dbModel.supports_functions),
    maxTokens: dbModel.max_tokens || dbModel.context_length || 0,
    rateMultiplier: dbModel.rate_multiplier ?? 1.0, // 从数据库获取倍率，使用nullish coalescing
    status: (dbModel.status === 'active' ? 'available' : dbModel.status === 'disabled' ? 'deprecated' : 'deprecated') as 'available' | 'beta' | 'deprecated',
    icon: getIcon(dbModel.model_type),
    color: dbModel.provider.color // 使用厂商的品牌颜色
  };



  return result;
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
            pricing: item.pricing,
            rate_multiplier: item.rate_multiplier,
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



  // 获取模型类型图标（用于筛选器）
  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'chat':
        return 'solar:chat-round-dots-bold';
      case 'image':
        return 'solar:eye-bold';
      case 'audio':
        return 'solar:share-bold';
      case 'embedding':
        return 'solar:pen-bold';
      default:
        return 'solar:pen-bold';
    }
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

      {/* 模型表格 */}
      <TableContainer component={Paper}>
        <Table sx={{ minWidth: 650 }}>
          <TableHead>
            <TableRow>
              <TableCell>{t('models.name', '模型名称')}</TableCell>
              <TableCell>{t('models.provider', '提供商')}</TableCell>
              <TableCell>{t('models.type', '类型')}</TableCell>
              <TableCell>{t('models.description', '描述')}</TableCell>
              <TableCell align="center">{t('models.context_length', '上下文长度')}</TableCell>
              <TableCell align="right">{t('models.pricing', '价格')}</TableCell>
              <TableCell align="right">{t('models.rate_multiplier', '倍率')}</TableCell>
              <TableCell align="center">{t('models.capabilities', '功能特性')}</TableCell>
              <TableCell align="center">{t('common.status', '状态')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredModels.map((model) => (
              <TableRow
                key={model.id}
                sx={{
                  '&:hover': {
                    backgroundColor: 'action.hover'
                  }
                }}
              >
                <TableCell>
                  <Typography variant="subtitle2" fontWeight="medium">
                    {model.name}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Chip
                    label={model.provider}
                    size="small"
                    sx={{
                      bgcolor: model.color,
                      color: 'white',
                      fontWeight: 'medium'
                    }}
                  />
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {model.type}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{
                      maxWidth: 300,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {model.description}
                  </Typography>
                </TableCell>
                <TableCell align="center">
                  {model.maxTokens > 0 ? (
                    <Typography variant="body2">
                      {model.maxTokens.toLocaleString()}
                    </Typography>
                  ) : (
                    <Typography variant="body2" color="text.secondary">
                      -
                    </Typography>
                  )}
                </TableCell>
                <TableCell align="right">
                  <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                    {model.pricing ? (
                      model.pricing.unit === 'image' || model.pricing.unit === 'request' ?
                        `$${model.pricing.input}/${model.pricing.unit}` :
                        model.type === 'text' ?
                          `$${model.pricing.input}/$${model.pricing.output}` :
                          `$${model.pricing.input}`
                    ) : '-'}
                  </Typography>
                </TableCell>
                <TableCell align="right">
                  <Typography variant="body2" sx={{ fontWeight: 'medium' }}>
                    {model.rateMultiplier ? `${model.rateMultiplier.toFixed(1)}x` : '1.0x'}
                  </Typography>
                </TableCell>
                <TableCell align="center">
                  <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', justifyContent: 'center' }}>
                    {model.capabilities.slice(0, 2).map((capability, index) => (
                      <Chip
                        key={index}
                        label={capability}
                        size="small"
                        variant="outlined"
                        color="primary"
                      />
                    ))}
                    {model.capabilities.length > 2 && (
                      <Chip
                        label={`+${model.capabilities.length - 2}`}
                        size="small"
                        variant="outlined"
                        color="default"
                      />
                    )}
                  </Box>
                </TableCell>
                <TableCell align="center">
                  <Chip
                    label={model.status}
                    size="small"
                    color={getStatusColor(model.status) as any}
                    variant="outlined"
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

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

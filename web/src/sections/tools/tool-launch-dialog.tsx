import { useState, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import Chip from '@mui/material/Chip';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import FormControl from '@mui/material/FormControl';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import InputLabel from '@mui/material/InputLabel';
import CircularProgress from '@mui/material/CircularProgress';

import { useAuthContext } from 'src/contexts/auth-context';
import { api } from 'src/services/api';

import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  status: string;
}

interface Model {
  id: number;
  name: string;
  slug: string;
  display_name: string;
  description: string;
  model_type: 'chat' | 'completion' | 'embedding' | 'image' | 'audio';
  provider: {
    id: number;
    name: string;
    display_name: string;
    color: string;
    sort_order: number;
  };
  context_length?: number;
  max_tokens?: number;
  supports_streaming: boolean;
  supports_functions: boolean;
  status: 'active' | 'deprecated' | 'disabled';
  created_at: string;
  updated_at: string;
}

interface Tool {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  supportedModels: string[];
}

type Props = {
  open: boolean;
  tool: Tool | null;
  onClose: () => void;
  onLaunch: (apiKey: string, model: string) => void;
};

// 从API获取的模型数据将在组件中动态加载

// ----------------------------------------------------------------------

export function ToolLaunchDialog({ open, tool, onClose, onLaunch }: Props) {
  const { t } = useTranslation();
  const { state } = useAuthContext();
  const [loading, setLoading] = useState(false);
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [selectedApiKey, setSelectedApiKey] = useState('');
  const [selectedModel, setSelectedModel] = useState<number | ''>('');
  const [estimatedCost, setEstimatedCost] = useState(0);

  // 获取用户的API Keys
  const fetchApiKeys = useCallback(async () => {
    if (!state.isAuthenticated) return;

    try {
      const response = await api.get('/admin/api-keys/');

      if (response.success && response.data) {
        const activeKeys = response.data.filter((key: ApiKey) => key.status === 'active');
        setApiKeys(activeKeys);
        if (activeKeys.length === 1) {
          setSelectedApiKey(activeKeys[0].id);
        }
      }
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
    }
  }, [state.isAuthenticated]);

  // 获取可用模型
  const fetchModels = useCallback(async () => {
    try {
      const response = await api.noAuth.get('/tools/models');

      if (response.success && response.data) {
        setModels(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch models:', error);
    }
  }, []);

  useEffect(() => {
    if (open) {
      setLoading(true);
      Promise.all([
        fetchApiKeys(),
        fetchModels()
      ]).finally(() => {
        setLoading(false);
      });
    }
  }, [open, fetchApiKeys, fetchModels]);

  // 根据工具类型筛选可用模型
  const getAvailableModels = useCallback(() => {
    if (!tool || !models.length) return [];

    return models.filter(model => {
      // 根据工具类型筛选模型
      if (tool.id === 'image-generator') {
        return model.model_type === 'image';
      }
      if (tool.id === 'chatbot') {
        return model.model_type === 'chat' || model.model_type === 'completion';
      }
      // 默认返回所有聊天模型
      return model.model_type === 'chat' || model.model_type === 'completion';
    });
  }, [tool, models]);

  const availableModels = getAvailableModels();

  const handleApiKeyChange = useCallback((event: any) => {
    setSelectedApiKey(event.target.value);
    setSelectedModel(''); // 重置模型选择
  }, []);

  const handleModelChange = useCallback((event: any) => {
    const modelId = Number(event.target.value);
    setSelectedModel(modelId);
    // 计算预估费用（简单示例）
    const model = availableModels.find(m => m.id === modelId);
    if (model) {
      // 使用默认价格，因为数据库模型可能没有定价信息
      setEstimatedCost(0.003 * 10); // 假设10K tokens，默认价格
    }
  }, [availableModels]);

  const handleLaunch = useCallback(() => {
    if (selectedApiKey && selectedModel) {
      onLaunch(selectedApiKey, selectedModel.toString());
      onClose();
    }
  }, [selectedApiKey, selectedModel, onLaunch, onClose]);

  const handleClose = useCallback(() => {
    setSelectedApiKey('');
    setSelectedModel('');
    setEstimatedCost(0);
    onClose();
  }, [onClose]);

  // 移除这行，因为event在这里不可用

  if (!tool) return null;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Box
            sx={{
              width: 48,
              height: 48,
              borderRadius: 2,
              bgcolor: tool.color,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Iconify 
              icon="solar:pen-bold"
              sx={{ width: 24, height: 24, color: 'white' }} 
            />
          </Box>
          <Box>
            <Typography variant="h6">
              {t('tools.launch_tool')}: {tool.name}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {tool.description}
            </Typography>
          </Box>
        </Box>
      </DialogTitle>

      <DialogContent>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* API Key 选择 */}
            <Box>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>
                {t('tools.select_api_key')}
              </Typography>
              
              {apiKeys.length === 0 ? (
                <Alert severity="warning">
                  {t('tools.no_api_keys_available')}
                  <Button 
                    size="small" 
                    sx={{ ml: 1 }}
                    onClick={() => window.open('/api-keys', '_blank')}
                  >
                    {t('tools.create_api_key')}
                  </Button>
                </Alert>
              ) : (
                <FormControl fullWidth>
                  <InputLabel>{t('tools.api_key')}</InputLabel>
                  <Select
                    value={selectedApiKey}
                    onChange={handleApiKeyChange}
                    label={t('tools.api_key')}
                  >
                    {apiKeys.map((apiKey) => (
                      <MenuItem key={apiKey.id} value={apiKey.id}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography>{apiKey.name}</Typography>
                          <Chip 
                            label={apiKey.key_prefix} 
                            size="small" 
                            variant="outlined"
                            sx={{ fontFamily: 'monospace' }}
                          />
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            </Box>

            {/* 模型选择 */}
            {selectedApiKey && (
              <Box>
                <Typography variant="subtitle1" sx={{ mb: 2 }}>
                  {t('tools.select_model')}
                </Typography>
                
                {availableModels.length === 0 ? (
                  <Alert severity="info">
                    {t('tools.no_compatible_models')}
                  </Alert>
                ) : (
                  <FormControl fullWidth>
                    <InputLabel>{t('tools.model')}</InputLabel>
                    <Select
                      value={selectedModel}
                      onChange={handleModelChange}
                      label={t('tools.model')}
                    >
                      {availableModels.map((model) => (
                        <MenuItem key={model.id} value={model.id}>
                          <Box sx={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                            <Box>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Typography>{model.display_name || model.name}</Typography>
                                <Chip
                                  label={model.provider.display_name}
                                  size="small"
                                  variant="outlined"
                                  sx={{
                                    height: 18,
                                    fontSize: '0.65rem',
                                    textTransform: 'capitalize',
                                    backgroundColor: model.provider.color + '20',
                                    borderColor: model.provider.color,
                                    color: model.provider.color
                                  }}
                                />
                              </Box>
                              <Typography variant="caption" color="text.secondary">
                                {model.model_type}
                              </Typography>
                            </Box>
                            <Typography variant="caption" color="primary.main">
                              {model.status}
                            </Typography>
                          </Box>
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
              </Box>
            )}

            {/* 模型详情和费用预估 */}
            {selectedModel && (
              <Card variant="outlined">
                <Box sx={{ p: 2 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    {t('tools.model_details')}
                  </Typography>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="body2" color="text.secondary">
                      {t('tools.model_type')}:
                    </Typography>
                    <Typography variant="body2">
                      {availableModels.find(m => m.id === selectedModel)?.model_type || 'Unknown'}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="body2" color="text.secondary">
                      {t('tools.status')}:
                    </Typography>
                    <Chip
                      label={availableModels.find(m => m.id === selectedModel)?.status || 'Unknown'}
                      size="small"
                      variant="outlined"
                      color={availableModels.find(m => m.id === selectedModel)?.status === 'active' ? 'success' : 'default'}
                    />
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="body2" color="text.secondary">
                      {t('tools.estimated_cost')}:
                    </Typography>
                    <Typography variant="body2" color="primary.main">
                      ${estimatedCost.toFixed(4)}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="body2" color="text.secondary">
                      {t('tools.current_balance')}:
                    </Typography>
                    <Typography variant="body2" color={state.user?.balance && state.user.balance > estimatedCost ? 'success.main' : 'error.main'}>
                      ${state.user?.balance?.toFixed(6) || '0.000000'}
                    </Typography>
                  </Box>
                </Box>
              </Card>
            )}

            {/* 余额不足警告 */}
            {selectedModel && state.user?.balance && state.user.balance < estimatedCost && (
              <Alert severity="error">
                {t('tools.insufficient_balance')}
                <Button 
                  size="small" 
                  sx={{ ml: 1 }}
                  onClick={() => window.open('/wallet', '_blank')}
                >
                  {t('tools.recharge_now')}
                </Button>
              </Alert>
            )}
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ px: 3, pb: 3 }}>
        <Button onClick={handleClose}>
          {t('common.cancel')}
        </Button>
        <Button
          variant="contained"
          onClick={handleLaunch}
          disabled={!selectedApiKey || !selectedModel || loading || Boolean(state.user?.balance && state.user.balance < estimatedCost)}
          startIcon={<Iconify icon="solar:play-bold" />}
        >
          {t('tools.launch_tool')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

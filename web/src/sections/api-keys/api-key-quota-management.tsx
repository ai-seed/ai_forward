import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import Chip from '@mui/material/Chip';
import Table from '@mui/material/Table';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import TableRow from '@mui/material/TableRow';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import DialogTitle from '@mui/material/DialogTitle';
import CardContent from '@mui/material/CardContent';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import TableContainer from '@mui/material/TableContainer';
import CircularProgress from '@mui/material/CircularProgress';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import IconButton from '@mui/material/IconButton';
import Alert from '@mui/material/Alert';

import api from 'src/services/api';

import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

interface Quota {
  id: number;
  quota_type: 'requests' | 'tokens' | 'cost';
  period?: 'minute' | 'hour' | 'day' | 'month' | null; // null表示总限额
  limit_value: number;
  used_value?: number;
  remaining?: number;
  reset_time?: string;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
}

interface ApiKeyQuotaManagementProps {
  apiKeyId: number;
}

interface CreateQuotaDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  apiKeyId: number;
}

// ----------------------------------------------------------------------

function CreateQuotaDialog({ open, onClose, onSuccess, apiKeyId }: CreateQuotaDialogProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [quotaType, setQuotaType] = useState<'requests' | 'tokens' | 'cost'>('requests');
  const [period, setPeriod] = useState<'minute' | 'hour' | 'day' | 'month' | 'total'>('day');
  const [limitValue, setLimitValue] = useState('');
  const [resetTime, setResetTime] = useState('00:00');

  const handleSubmit = async () => {
    if (!limitValue || parseFloat(limitValue) <= 0) {
      return;
    }

    try {
      setLoading(true);
      const token = localStorage.getItem('access_token');
      if (!token) return;

      const requestData = {
        quota_type: quotaType,
        period: period === 'total' ? null : period,
        limit_value: parseFloat(limitValue),
        reset_time: period === 'day' || period === 'month' ? resetTime : null,
      };

      const response = await api.post(`/admin/api-keys/${apiKeyId}/quotas`, requestData);

      if (response.success) {
        onSuccess();
        onClose();
        // 重置表单
        setQuotaType('requests');
        setPeriod('day');
        setLimitValue('');
        setResetTime('00:00');
      }
    } catch (error) {
      console.error('Error creating quota:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t('quota.create_quota')}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 1 }}>
          <FormControl fullWidth>
            <InputLabel>{t('quota.quota_type')}</InputLabel>
            <Select
              value={quotaType}
              label={t('quota.quota_type')}
              onChange={(e) => setQuotaType(e.target.value as any)}
            >
              <MenuItem value="requests">{t('quota.quota_types.requests')}</MenuItem>
              <MenuItem value="tokens">{t('quota.quota_types.tokens')}</MenuItem>
              <MenuItem value="cost">{t('quota.quota_types.cost')}</MenuItem>
            </Select>
          </FormControl>

          <FormControl fullWidth>
            <InputLabel>{t('quota.period_type')}</InputLabel>
            <Select
              value={period}
              label={t('quota.period_type')}
              onChange={(e) => setPeriod(e.target.value as any)}
            >
              <MenuItem value="total">{t('quota.periods.total')}</MenuItem>
              <MenuItem value="minute">{t('quota.periods.minute')}</MenuItem>
              <MenuItem value="hour">{t('quota.periods.hour')}</MenuItem>
              <MenuItem value="day">{t('quota.periods.day')}</MenuItem>
              <MenuItem value="month">{t('quota.periods.month')}</MenuItem>
            </Select>
          </FormControl>

          <TextField
            fullWidth
            label={t('quota.limit_value')}
            type="number"
            value={limitValue}
            onChange={(e) => setLimitValue(e.target.value)}
            helperText={
              quotaType === 'requests' ? t('quota.quota_types.requests') :
              quotaType === 'tokens' ? t('quota.quota_types.tokens') :
              t('quota.quota_types.cost')
            }
          />

          {(period === 'day' || period === 'month') && (
            <TextField
              fullWidth
              label={t('quota.reset_time')}
              type="time"
              value={resetTime}
              onChange={(e) => setResetTime(e.target.value)}
              helperText={t('quota.helper_text.total_quota')}
              InputLabelProps={{
                shrink: true,
              }}
            />
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={loading || !limitValue}
        >
          {loading ? <CircularProgress size={20} /> : t('common.create')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

// ----------------------------------------------------------------------

export function ApiKeyQuotaManagement({ apiKeyId }: ApiKeyQuotaManagementProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [quotas, setQuotas] = useState<Quota[]>([]);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const fetchQuotas = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('access_token');
      if (!token) return;

      const response = await api.get(`/admin/api-keys/${apiKeyId}/quotas`);

      if (response.success && response.data) {
        setQuotas(response.data);
      }
    } catch (error) {
      console.error('Error fetching quotas:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (apiKeyId) {
      fetchQuotas();
    }
  }, [apiKeyId]);

  const handleDeleteQuota = async (quotaId: number) => {
    if (!confirm('确定要删除这个配额限制吗？')) {
      return;
    }

    try {
      const token = localStorage.getItem('access_token');
      if (!token) return;

      const response = await api.delete(`/admin/quotas/${quotaId}`);

      if (response.success) {
        // 直接从本地状态中移除该配额，无需重新获取整个列表
        setQuotas(prevQuotas => prevQuotas.filter(quota => quota.id !== quotaId));
      }
    } catch (error) {
      console.error('Error deleting quota:', error);
    }
  };

  const handleToggleStatus = async (quotaId: number, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'inactive' : 'active';

    try {
      const token = localStorage.getItem('access_token');
      if (!token) return;

      const response = await api.put(`/admin/quotas/${quotaId}`, {
        status: newStatus,
      });

      if (response.success) {
        // 直接更新本地状态中的配额状态，无需重新获取整个列表
        setQuotas(prevQuotas =>
          prevQuotas.map(quota =>
            quota.id === quotaId ? { ...quota, status: newStatus } : quota
          )
        );
      }
    } catch (error) {
      console.error('Error updating quota status:', error);
    }
  };

  const getQuotaTypeLabel = (type: string) => {
    switch (type) {
      case 'requests': return t('quota.quota_types.requests');
      case 'tokens': return t('quota.quota_types.tokens');
      case 'cost': return t('quota.quota_types.cost');
      default: return type;
    }
  };

  const getPeriodLabel = (period: string | null | undefined) => {
    if (!period) return t('quota.periods.total');
    switch (period) {
      case 'minute': return t('quota.periods.minute');
      case 'hour': return t('quota.periods.hour');
      case 'day': return t('quota.periods.day');
      case 'month': return t('quota.periods.month');
      default: return period;
    }
  };

  const formatValue = (type: string, value: number) => {
    if (type === 'cost') {
      return `$${value.toFixed(6)}`;
    }
    return value.toLocaleString();
  };

  const getUsagePercentage = (used: number = 0, limit: number) => {
    return Math.min((used / limit) * 100, 100);
  };

  const getStatusColor = (status: string) => {
    return status === 'active' ? 'success' : 'default';
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'active': return t('quota.status.active');
      case 'inactive': return t('quota.status.inactive');
      default: return status;
    }
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h6">{t('quota.management')}</Typography>
        <Button
          variant="contained"
          startIcon={<Iconify icon="solar:check-circle-bold" />}
          onClick={() => setCreateDialogOpen(true)}
        >
          {t('quota.add_quota')}
        </Button>
      </Box>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
          <CircularProgress />
        </Box>
      ) : quotas.length === 0 ? (
        <Alert severity="info">
          {t('quota.no_quota_limits')}
        </Alert>
      ) : (
        <TableContainer component={Card}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('quota.quota_type')}</TableCell>
                <TableCell>{t('quota.period')}</TableCell>
                <TableCell>{t('quota.limit_value')}</TableCell>
                <TableCell>{t('quota.used_value')}</TableCell>
                <TableCell>{t('quota.usage_rate')}</TableCell>
                <TableCell>{t('common.status')}</TableCell>
                <TableCell>{t('common.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {quotas.map((quota) => (
                <TableRow key={quota.id}>
                  <TableCell>{getQuotaTypeLabel(quota.quota_type)}</TableCell>
                  <TableCell>{getPeriodLabel(quota.period)}</TableCell>
                  <TableCell>{formatValue(quota.quota_type, quota.limit_value)}</TableCell>
                  <TableCell>{formatValue(quota.quota_type, quota.used_value || 0)}</TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Box
                        sx={{
                          width: 60,
                          height: 8,
                          bgcolor: 'grey.300',
                          borderRadius: 1,
                          overflow: 'hidden',
                        }}
                      >
                        <Box
                          sx={{
                            width: `${getUsagePercentage(quota.used_value, quota.limit_value)}%`,
                            height: '100%',
                            bgcolor: getUsagePercentage(quota.used_value, quota.limit_value) > 80 ? 'error.main' : 'primary.main',
                          }}
                        />
                      </Box>
                      <Typography variant="body2">
                        {getUsagePercentage(quota.used_value, quota.limit_value).toFixed(1)}%
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getStatusLabel(quota.status)}
                      color={getStatusColor(quota.status) as any}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <IconButton
                        size="small"
                        onClick={() => handleToggleStatus(quota.id, quota.status)}
                        color={quota.status === 'active' ? 'warning' : 'success'}
                      >
                        <Iconify icon={quota.status === 'active' ? 'solar:pause-bold' : 'solar:play-bold'} />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => handleDeleteQuota(quota.id)}
                        color="error"
                      >
                        <Iconify icon="solar:trash-bin-trash-bold" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <CreateQuotaDialog
        open={createDialogOpen}
        onClose={() => setCreateDialogOpen(false)}
        onSuccess={fetchQuotas}
        apiKeyId={apiKeyId}
      />
    </Box>
  );
}

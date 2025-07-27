import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import InputAdornment from '@mui/material/InputAdornment';
import CircularProgress from '@mui/material/CircularProgress';

import { useRouter } from 'src/routes/hooks';

import AuthService from 'src/services/auth';

import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

export function ResetPasswordView() {
  const { t } = useTranslation();
  const router = useRouter();

  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  // 邮箱验证码相关状态
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [sendingCode, setSendingCode] = useState(false);

  // 表单数据
  const [formData, setFormData] = useState({
    email: '',
    verificationCode: '',
    newPassword: '',
    confirmPassword: '',
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [resetSuccess, setResetSuccess] = useState(false);

  // 倒计时效果
  const startCountdown = useCallback(() => {
    setCountdown(60);
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, []);

  // 发送验证码
  const handleSendCode = useCallback(async () => {
    if (!formData.email.trim()) {
      setFormErrors(prev => ({ ...prev, email: t('auth.email_required') }));
      return;
    }

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      setFormErrors(prev => ({ ...prev, email: t('auth.email_invalid') }));
      return;
    }

    setSendingCode(true);
    setError('');

    try {
      await AuthService.sendVerificationCode({
        email: formData.email.trim(),
        type: 'password_reset'
      });

      setCodeSent(true);
      startCountdown();
      setError('');
      // 清除邮箱错误
      setFormErrors(prev => ({ ...prev, email: '' }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to send verification code';
      setError(errorMessage);
    } finally {
      setSendingCode(false);
    }
  }, [formData.email, t, startCountdown]);



  const handleInputChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = event.target;

    setFormData(prev => ({
      ...prev,
      [name]: value,
    }));

    // 清除对应字段的错误
    if (formErrors[name]) {
      setFormErrors(prev => ({
        ...prev,
        [name]: '',
      }));
    }

    // 清除全局错误信息
    if (error) {
      setError('');
    }
  }, [formErrors, error]);

  const validateForm = useCallback(() => {
    const errors: Record<string, string> = {};

    if (!formData.email.trim()) {
      errors.email = t('auth.email_required');
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      errors.email = t('auth.email_invalid');
    }

    if (!formData.verificationCode.trim()) {
      errors.verificationCode = t('auth.verification_code_required');
    } else if (formData.verificationCode.length !== 6) {
      errors.verificationCode = t('auth.verification_code_invalid');
    }

    if (!formData.newPassword) {
      errors.newPassword = t('auth.password_required');
    } else if (formData.newPassword.length < 6) {
      errors.newPassword = t('auth.password_min_length');
    }

    if (!formData.confirmPassword) {
      errors.confirmPassword = t('auth.confirm_password_required');
    } else if (formData.newPassword !== formData.confirmPassword) {
      errors.confirmPassword = t('auth.passwords_not_match');
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  }, [formData, t]);

  const handleResetPassword = useCallback(async (event: React.FormEvent) => {
    event.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);
    setError('');

    try {
      await AuthService.resetPassword({
        email: formData.email.trim(),
        new_password: formData.newPassword,
        verification_code: formData.verificationCode.trim(),
      });

      // 重置成功
      setResetSuccess(true);
    } catch (error) {
      let errorMessage = 'Password reset failed';

      if (error instanceof Error) {
        const errorCode = (error as any).code;

        // 根据错误代码显示相应的翻译文本
        switch (errorCode) {
          case 'VERIFICATION_CODE_ERROR':
            errorMessage = t('auth.verification_code_error');
            break;
          default:
            errorMessage = error.message;
        }
      }

      setError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, [formData, validateForm, t]);

  const handleGoToSignIn = useCallback(() => {
    router.push('/sign-in');
  }, [router]);



  if (resetSuccess) {
    return (
      <Box sx={{ textAlign: 'center', p: 3 }}>
        <Typography variant="h4" sx={{ mb: 2 }}>
          {t('auth.password_reset_successful')}
        </Typography>
        <Typography variant="body1" sx={{ mb: 3, color: 'text.secondary' }}>
          {t('auth.password_reset_complete')}
        </Typography>
        <Button
          variant="contained"
          size="large"
          onClick={handleGoToSignIn}
        >
          {t('auth.go_to_signin')}
        </Button>
      </Box>
    );
  }



  // 渲染重置密码表单
  const renderForm = (
    <Box
      component="form"
      onSubmit={handleResetPassword}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      {error && (
        <Alert severity="error">
          {error}
        </Alert>
      )}

      <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
        <TextField
          fullWidth
          name="email"
          label={t('auth.email')}
          type="email"
          value={formData.email}
          onChange={handleInputChange}
          disabled={isLoading}
          required
          error={!!formErrors.email}
          helperText={formErrors.email}
          slotProps={{
            inputLabel: { shrink: true },
          }}
        />
        <Button
          variant="outlined"
          onClick={handleSendCode}
          disabled={sendingCode || countdown > 0 || !formData.email.trim()}
          sx={{
            minWidth: 120,
            height: 56, // 匹配TextField高度
            whiteSpace: 'nowrap'
          }}
          startIcon={sendingCode ? <CircularProgress size={16} /> : null}
        >
          {sendingCode
            ? t('common.loading')
            : countdown > 0
              ? `${countdown}s`
              : codeSent
                ? t('auth.resend_code')
                : t('auth.send_code')
          }
        </Button>
      </Box>

      <TextField
        fullWidth
        name="verificationCode"
        label={t('auth.verification_code')}
        value={formData.verificationCode}
        onChange={handleInputChange}
        disabled={isLoading}
        required
        error={!!formErrors.verificationCode}
        helperText={formErrors.verificationCode}
        inputProps={{ maxLength: 6 }}
        slotProps={{
          inputLabel: { shrink: true },
        }}
      />

      <TextField
        fullWidth
        name="newPassword"
        label={t('auth.new_password')}
        value={formData.newPassword}
        onChange={handleInputChange}
        disabled={isLoading}
        required
        error={!!formErrors.newPassword}
        helperText={formErrors.newPassword}
        type={showPassword ? 'text' : 'password'}
        slotProps={{
          inputLabel: { shrink: true },
          input: {
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  onClick={() => setShowPassword(!showPassword)}
                  edge="end"
                  disabled={isLoading}
                >
                  <Iconify icon={showPassword ? 'solar:eye-bold' : 'solar:eye-closed-bold'} />
                </IconButton>
              </InputAdornment>
            ),
          },
        }}
      />

      <TextField
        fullWidth
        name="confirmPassword"
        label={t('auth.confirm_password')}
        value={formData.confirmPassword}
        onChange={handleInputChange}
        disabled={isLoading}
        required
        error={!!formErrors.confirmPassword}
        helperText={formErrors.confirmPassword}
        type={showConfirmPassword ? 'text' : 'password'}
        slotProps={{
          inputLabel: { shrink: true },
          input: {
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  edge="end"
                  disabled={isLoading}
                >
                  <Iconify icon={showConfirmPassword ? 'solar:eye-bold' : 'solar:eye-closed-bold'} />
                </IconButton>
              </InputAdornment>
            ),
          },
        }}
      />

      <Button
        fullWidth
        size="large"
        type="submit"
        color="inherit"
        variant="contained"
        disabled={isLoading}
        startIcon={isLoading ? <CircularProgress size={20} /> : null}
      >
        {isLoading ? t('common.loading') : t('auth.reset_password')}
      </Button>
    </Box>
  );

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h4" sx={{ mb: 1.5 }}>
        {t('auth.reset_password')}
      </Typography>

      <Typography variant="body2" sx={{ color: 'text.secondary', mb: 3 }}>
        {t('auth.enter_email_and_new_password')}
      </Typography>

      {renderForm}

      <Box sx={{ mt: 2, textAlign: 'center' }}>
        <Link
          variant="subtitle2"
          sx={{ cursor: 'pointer' }}
          onClick={handleGoToSignIn}
        >
          {t('auth.back_to_signin')}
        </Link>
      </Box>
    </Box>
  );
}

import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { Box, Typography, CircularProgress } from '@mui/material';

import AuthService from 'src/services/auth';
import { useAuth } from 'src/contexts/auth-context';

// ----------------------------------------------------------------------

export default function OAuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { handleOAuthCallback } = useAuth();

  useEffect(() => {
    const handleCallback = async () => {
      // 检查是否有直接的token参数（来自后端重定向）
      const accessToken = searchParams.get('access_token');
      const refreshToken = searchParams.get('refresh_token');

      if (accessToken && refreshToken) {
        // 直接从URL参数获取token
        try {
          // 存储token到localStorage
          localStorage.setItem('access_token', accessToken);
          localStorage.setItem('refresh_token', refreshToken);

          // 获取用户信息
          const userInfo = await AuthService.getProfile();
          localStorage.setItem('user_info', JSON.stringify(userInfo));

          // 跳转到首页
          navigate('/', { replace: true });
          return;
        } catch (error) {
          console.error('Failed to process OAuth tokens:', error);
          navigate('/sign-in', {
            replace: true,
            state: { error: 'OAuth login failed' }
          });
          return;
        }
      }

      // 如果没有token参数，说明可能是错误的回调
      console.error('Missing OAuth tokens in callback URL');
      navigate('/sign-in', {
        replace: true,
        state: { error: 'OAuth callback failed: missing tokens' }
      });
      return;
    };

    handleCallback();
  }, [searchParams, navigate, handleOAuthCallback]);

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        gap: 2,
      }}
    >
      <CircularProgress size={40} />
      <Typography variant="body1" color="text.secondary">
        Processing OAuth login...
      </Typography>
    </Box>
  );
}

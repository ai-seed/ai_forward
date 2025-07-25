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

          // 跳转到仪表板
          navigate('/dashboard', { replace: true });
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

      // 原有的code/state处理逻辑（备用）
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const provider = window.location.pathname.split('/')[3]; // 从路径中提取provider

      if (!code || !state || !provider) {
        console.error('Missing OAuth callback parameters');
        navigate('/sign-in', {
          replace: true,
          state: { error: 'OAuth callback failed: missing parameters' }
        });
        return;
      }

      try {
        await handleOAuthCallback({
          provider,
          code,
          state,
        });

        // 登录成功，重定向到仪表板
        navigate('/dashboard', { replace: true });
      } catch (error) {
        console.error('OAuth callback failed:', error);
        navigate('/sign-in', {
          replace: true,
          state: { error: 'OAuth login failed' }
        });
      }
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

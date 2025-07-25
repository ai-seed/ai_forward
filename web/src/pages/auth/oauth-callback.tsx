import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { Box, Typography, CircularProgress } from '@mui/material';

import { useAuth } from 'src/contexts/auth-context';

// ----------------------------------------------------------------------

export default function OAuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { handleOAuthCallback } = useAuth();

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const provider = window.location.pathname.split('/')[3]; // 从路径中提取provider

      if (!code || !state || !provider) {
        console.error('Missing OAuth callback parameters');
        navigate('/auth/login', { 
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
        navigate('/auth/login', { 
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

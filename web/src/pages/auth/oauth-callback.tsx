import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

import { Box, Typography, CircularProgress } from '@mui/material';

// ----------------------------------------------------------------------

export default function OAuthCallbackPage() {
  const navigate = useNavigate();

  useEffect(() => {
    // OAuth回调页面直接重定向到首页
    // token处理由App组件的useOAuthTokenHandler处理
    navigate('/', { replace: true });
  }, [navigate]);

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

import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

import { Box, Typography, CircularProgress } from '@mui/material';

// ----------------------------------------------------------------------

export default function OAuthCallbackPage() {
  const navigate = useNavigate();

  useEffect(() => {
    // 这个页面现在不应该被访问到，因为后端直接重定向到首页
    // 如果用户到了这里，说明可能有问题，重定向到登录页
    navigate('/sign-in', { replace: true });
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

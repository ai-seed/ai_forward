import type { ReactNode } from 'react';

import { useEffect, useState } from 'react';

import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import CircularProgress from '@mui/material/CircularProgress';

import { useRouter } from 'src/routes/hooks';

import { useAuth } from 'src/contexts/auth-context';

interface ProtectedRouteProps {
  children: ReactNode;
  fallback?: ReactNode;
}

/**
 * 路由守卫组件
 * 保护需要认证的路由，未登录用户会被重定向到登录页
 */
export function ProtectedRoute({ children, fallback }: ProtectedRouteProps) {
  const { state } = useAuth();
  const router = useRouter();

  useEffect(() => {
    // 检查是否有OAuth token在URL中
    const urlParams = new URLSearchParams(window.location.search);
    const hasOAuthTokens = urlParams.get('access_token') && urlParams.get('refresh_token');

    // 如果有OAuth tokens，不要重定向，让OAuth处理完成
    if (hasOAuthTokens) {
      console.log('🔄 OAuth tokens detected, waiting for processing...');
      return;
    }

    // 如果认证检查完成且用户未登录，重定向到登录页
    if (!state.isLoading && !state.isAuthenticated) {
      console.log('🔄 Redirecting to login page');
      router.replace('/sign-in');
    }
  }, [state.isLoading, state.isAuthenticated, router]);

  // 如果正在加载认证状态，显示加载指示器
  if (state.isLoading) {
    return (
      fallback || (
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
          <Typography variant="body2" color="text.secondary">
            Loading...
          </Typography>
        </Box>
      )
    );
  }

  // 如果用户已认证，渲染子组件
  if (state.isAuthenticated) {
    return <>{children}</>;
  }

  // 如果用户未认证，不渲染任何内容（将被重定向）
  return null;
}

export default ProtectedRoute;

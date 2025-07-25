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
  const [shouldRedirect, setShouldRedirect] = useState(false);

  useEffect(() => {
    // 检查是否有OAuth token在URL中
    const urlParams = new URLSearchParams(window.location.search);
    const hasOAuthTokens = urlParams.get('access_token') && urlParams.get('refresh_token');

    if (!state.isLoading && !state.isAuthenticated) {
      if (hasOAuthTokens) {
        // 如果有OAuth tokens，给OAuth处理一些时间
        console.log('🔄 OAuth tokens detected, delaying redirect...');
        const timer = setTimeout(() => {
          // 再次检查认证状态
          if (!state.isAuthenticated) {
            console.log('⏰ OAuth processing timeout, redirecting to login');
            setShouldRedirect(true);
          }
        }, 2000); // 给OAuth处理2秒时间

        return () => clearTimeout(timer);
      } else {
        // 没有OAuth tokens，立即重定向
        setShouldRedirect(true);
      }
    } else if (state.isAuthenticated) {
      // 如果已认证，取消重定向
      setShouldRedirect(false);
    }

    return undefined;
  }, [state.isLoading, state.isAuthenticated]);

  useEffect(() => {
    if (shouldRedirect) {
      console.log('🔄 Redirecting to login page');
      router.replace('/sign-in');
    }
  }, [shouldRedirect, router]);

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

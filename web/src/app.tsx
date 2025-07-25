import 'src/global.css';

import { useEffect } from 'react';

import { usePathname } from 'src/routes/hooks';

import TokenStorage from 'src/utils/token-storage';

import AuthService from 'src/services/auth';
import { ThemeProvider } from 'src/theme/theme-provider';
import { AuthProvider, useAuth } from 'src/contexts/auth-context';

// ----------------------------------------------------------------------

type AppProps = {
  children: React.ReactNode;
};

export default function App({ children }: AppProps) {
  useScrollToTop();
  useOAuthTokenHandler();

  return (
    <ThemeProvider>
      <AuthProvider>
        {children}
      </AuthProvider>
    </ThemeProvider>
  );
}

// ----------------------------------------------------------------------

function useScrollToTop() {
  const pathname = usePathname();

  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);

  return null;
}

function useOAuthTokenHandler() {
  useEffect(() => {
    const handleOAuthTokens = async () => {
      // 检查URL参数中是否有OAuth token
      const urlParams = new URLSearchParams(window.location.search);
      const accessToken = urlParams.get('access_token');
      const refreshToken = urlParams.get('refresh_token');

      if (accessToken && refreshToken) {
        try {
          // 存储token
          TokenStorage.setAccessToken(accessToken);
          TokenStorage.setRefreshToken(refreshToken);

          // 使用项目的AuthService获取用户信息
          const userInfo = await AuthService.getProfile();
          // 存储用户信息
          TokenStorage.setUserInfo(userInfo);

          // 清除URL参数
          const newUrl = window.location.pathname;
          window.history.replaceState({}, document.title, newUrl);

          // 触发一个自定义事件，通知认证状态更新
          window.dispatchEvent(new CustomEvent('oauth-login-success', {
            detail: { userInfo }
          }));

        } catch (error) {
          console.error('Failed to process OAuth tokens:', error);
          // 如果出错，清除可能的无效token
          TokenStorage.clearAuthData();
          // 重定向到登录页
          window.location.href = '/sign-in';
        }
      }
    };

    handleOAuthTokens();
  }, []);

  return null;
}

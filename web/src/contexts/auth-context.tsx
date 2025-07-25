import type { ReactNode } from 'react';

import { useTranslation } from 'react-i18next';
import React, { useEffect, useContext, useReducer, createContext } from 'react';

import AuthService from '../services/auth';

import type { UserInfo, LoginRequest, RegisterRequest, ChangePasswordRequest, OAuthLoginRequest } from '../services/auth';

// 认证状态类型
export interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: UserInfo | null;
  error: string | null;
}

// 认证动作类型
export type AuthAction =
  | { type: 'AUTH_START' }
  | { type: 'AUTH_SUCCESS'; payload: UserInfo }
  | { type: 'AUTH_FAILURE'; payload: string }
  | { type: 'AUTH_LOGOUT' }
  | { type: 'AUTH_CLEAR_ERROR' }
  | { type: 'AUTH_UPDATE_USER'; payload: UserInfo };

// 认证上下文类型
export interface AuthContextType {
  state: AuthState;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => void;
  changePassword: (passwordData: ChangePasswordRequest) => Promise<void>;
  clearError: () => void;
  checkAuth: () => void;
  oauthLogin: (provider: string) => Promise<void>;
  handleOAuthCallback: (request: OAuthLoginRequest) => Promise<void>;
}

// 初始状态
const initialState: AuthState = {
  isAuthenticated: false,
  isLoading: true,
  user: null,
  error: null,
};

// 认证状态reducer
function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'AUTH_START':
      return {
        ...state,
        isLoading: true,
        error: null,
      };
    case 'AUTH_SUCCESS':
      return {
        ...state,
        isAuthenticated: true,
        isLoading: false,
        user: action.payload,
        error: null,
      };
    case 'AUTH_FAILURE':
      return {
        ...state,
        isAuthenticated: false,
        isLoading: false,
        user: null,
        error: action.payload,
      };
    case 'AUTH_LOGOUT':
      return {
        ...state,
        isAuthenticated: false,
        isLoading: false,
        user: null,
        error: null,
      };
    case 'AUTH_CLEAR_ERROR':
      return {
        ...state,
        error: null,
      };
    case 'AUTH_UPDATE_USER':
      return {
        ...state,
        user: action.payload,
      };
    default:
      return state;
  }
}

// 创建认证上下文
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// 认证提供者组件
export function AuthProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const [state, dispatch] = useReducer(authReducer, initialState);

  // 检查认证状态
  const checkAuth = () => {
    console.log('🔍 Checking auth state...');
    dispatch({ type: 'AUTH_START' });

    try {
      const isAuthenticated = AuthService.isAuthenticated();
      const user = AuthService.getCurrentUser();

      console.log('🔐 Auth check results:', { isAuthenticated, user: !!user });

      if (isAuthenticated && user) {
        console.log('✅ User authenticated, dispatching AUTH_SUCCESS');
        dispatch({ type: 'AUTH_SUCCESS', payload: user });

        // 自动刷新token（如果需要）
        AuthService.autoRefreshToken().catch((error) => {
          console.error('Auto refresh failed:', error);
          dispatch({ type: 'AUTH_LOGOUT' });
        });
      } else {
        console.log('❌ User not authenticated, dispatching AUTH_LOGOUT');
        dispatch({ type: 'AUTH_LOGOUT' });
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      dispatch({ type: 'AUTH_LOGOUT' });
    }
  };

  // 用户登录
  const login = async (credentials: LoginRequest) => {
    dispatch({ type: 'AUTH_START' });
    
    try {
      const response = await AuthService.login(credentials);
      dispatch({ type: 'AUTH_SUCCESS', payload: response.user });
    } catch (error) {
      let errorMessage = 'Login failed';

      if (error instanceof Error) {
        // 处理特殊的错误码
        if (error.message === 'INVALID_CREDENTIALS') {
          errorMessage = t('auth.invalid_credentials_error');
        } else {
          errorMessage = error.message;
        }
      }

      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      throw error;
    }
  };

  // 用户注册
  const register = async (userData: RegisterRequest) => {
    dispatch({ type: 'AUTH_START' });
    
    try {
      await AuthService.register(userData);
      // 注册成功后不自动登录，让用户手动登录
      dispatch({ type: 'AUTH_LOGOUT' });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Registration failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      throw error;
    }
  };

  // 用户登出
  const logout = () => {
    AuthService.logout();
    dispatch({ type: 'AUTH_LOGOUT' });
  };

  // 修改密码
  const changePassword = async (passwordData: ChangePasswordRequest) => {
    try {
      await AuthService.changePassword(passwordData);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Password change failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      throw error;
    }
  };

  // 清除错误
  const clearError = () => {
    dispatch({ type: 'AUTH_CLEAR_ERROR' });
  };

  // OAuth登录
  const oauthLogin = async (provider: string) => {
    dispatch({ type: 'AUTH_START' });

    try {
      await AuthService.oauthLogin(provider);
      // 重定向会在AuthService中处理
    } catch (error) {
      let errorMessage = 'OAuth login failed';

      if (error instanceof Error) {
        errorMessage = error.message;
      }

      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      throw error;
    }
  };

  // 处理OAuth回调
  const handleOAuthCallback = async (request: OAuthLoginRequest) => {
    dispatch({ type: 'AUTH_START' });

    try {
      const response = await AuthService.handleOAuthCallback(request);
      dispatch({ type: 'AUTH_SUCCESS', payload: response.user });
    } catch (error) {
      let errorMessage = 'OAuth callback failed';

      if (error instanceof Error) {
        if (error.message === 'OAUTH_FAILED') {
          errorMessage = t('auth.oauth_failed_error');
        } else {
          errorMessage = error.message;
        }
      }

      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      throw error;
    }
  };

  // 组件挂载时检查认证状态
  useEffect(() => {
    checkAuth();
  }, []);

  // 定期检查token是否需要刷新
  useEffect(() => {
    if (state.isAuthenticated) {
      const interval = setInterval(() => {
        AuthService.autoRefreshToken().catch((error) => {
          console.error('Auto refresh failed:', error);
          logout();
        });
      }, 5 * 60 * 1000); // 每5分钟检查一次

      return () => clearInterval(interval);
    }
    return undefined; // 明确返回undefined
  }, [state.isAuthenticated]);

  const contextValue: AuthContextType = {
    state,
    login,
    register,
    logout,
    changePassword,
    clearError,
    checkAuth,
    oauthLogin,
    handleOAuthCallback,
  };

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
}

// 使用认证上下文的Hook
export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

// 别名导出，保持向后兼容
export const useAuthContext = useAuth;

export default AuthContext;

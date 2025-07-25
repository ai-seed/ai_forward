/**
 * Token存储管理工具
 * 统一管理localStorage中的认证相关数据
 */

// 存储键名常量
const STORAGE_KEYS = {
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',
  USER_INFO: 'user_info',
} as const;

// 使用通用的用户信息类型
export interface UserInfo {
  id: number;
  username: string;
  email: string;
  full_name?: string;
  balance?: number;
  avatar?: string;
  status?: string;
  auth_method?: string;
  created_at?: string;
  updated_at?: string;
}

/**
 * Token存储管理类
 */
export class TokenStorage {
  /**
   * 获取访问令牌
   */
  static getAccessToken(): string | null {
    return localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
  }

  /**
   * 设置访问令牌
   */
  static setAccessToken(token: string): void {
    localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, token);
  }

  /**
   * 移除访问令牌
   */
  static removeAccessToken(): void {
    localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
  }

  /**
   * 获取刷新令牌
   */
  static getRefreshToken(): string | null {
    return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
  }

  /**
   * 设置刷新令牌
   */
  static setRefreshToken(token: string): void {
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, token);
  }

  /**
   * 移除刷新令牌
   */
  static removeRefreshToken(): void {
    localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  }

  /**
   * 获取用户信息
   */
  static getUserInfo(): UserInfo | null {
    try {
      const userInfo = localStorage.getItem(STORAGE_KEYS.USER_INFO);
      return userInfo ? JSON.parse(userInfo) : null;
    } catch (error) {
      console.error('Failed to parse user info from localStorage:', error);
      return null;
    }
  }

  /**
   * 设置用户信息
   */
  static setUserInfo(userInfo: UserInfo): void {
    localStorage.setItem(STORAGE_KEYS.USER_INFO, JSON.stringify(userInfo));
  }

  /**
   * 移除用户信息
   */
  static removeUserInfo(): void {
    localStorage.removeItem(STORAGE_KEYS.USER_INFO);
  }

  /**
   * 设置完整的认证数据
   */
  static setAuthData(accessToken: string, refreshToken: string, userInfo: UserInfo): void {
    this.setAccessToken(accessToken);
    this.setRefreshToken(refreshToken);
    this.setUserInfo(userInfo);
  }

  /**
   * 获取完整的认证数据
   */
  static getAuthData(): {
    accessToken: string | null;
    refreshToken: string | null;
    userInfo: UserInfo | null;
  } {
    return {
      accessToken: this.getAccessToken(),
      refreshToken: this.getRefreshToken(),
      userInfo: this.getUserInfo(),
    };
  }

  /**
   * 清除所有认证数据
   */
  static clearAuthData(): void {
    this.removeAccessToken();
    this.removeRefreshToken();
    this.removeUserInfo();
  }

  /**
   * 检查是否有有效的认证数据
   */
  static hasValidAuthData(): boolean {
    const accessToken = this.getAccessToken();
    const userInfo = this.getUserInfo();
    return !!(accessToken && userInfo);
  }

  /**
   * 检查访问令牌是否存在
   */
  static hasAccessToken(): boolean {
    return !!this.getAccessToken();
  }

  /**
   * 检查刷新令牌是否存在
   */
  static hasRefreshToken(): boolean {
    return !!this.getRefreshToken();
  }

  /**
   * 检查用户信息是否存在
   */
  static hasUserInfo(): boolean {
    return !!this.getUserInfo();
  }
}

// 导出默认实例和常量
export default TokenStorage;
export { STORAGE_KEYS };

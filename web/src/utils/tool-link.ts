import { CONFIG } from '../config-global';

// 工具实例接口
export interface ToolInstance {
  id: string;
  name: string;
  description?: string;
  code: string;
  tool?: {
    id: string;
    name: string;
    path?: string;
  };
}

/**
 * 生成工具链接
 * @param toolInstance 工具实例
 * @returns 工具链接URL
 */
export function generateToolLink(toolInstance: ToolInstance): string {
  if (!toolInstance.code) {
    throw new Error('Tool instance code is required');
  }

  if (!toolInstance.tool?.path) {
    throw new Error('Tool path is not configured');
  }

  const baseUrl = CONFIG.toolBaseUrl;
  const params = new URLSearchParams({
    code: toolInstance.code,
    to: toolInstance.tool.path,
  });

  return `${baseUrl}?${params.toString()}`;
}

/**
 * 检查工具实例是否可以生成链接
 * @param toolInstance 工具实例
 * @returns 是否可以生成链接
 */
export function canGenerateToolLink(toolInstance: ToolInstance): boolean {
  return !!(toolInstance.code && toolInstance.tool?.path);
}

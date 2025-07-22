import packageJson from '../package.json';

// ----------------------------------------------------------------------

export type ConfigValue = {
  appName: string;
  appVersion: string;
  toolBaseUrl: string;
};

export const CONFIG: ConfigValue = {
  appName: 'Minimal UI',
  appVersion: packageJson.version,
  toolBaseUrl: import.meta.env.VITE_TOOL_BASE_URL || 'https://tools-dev.718ai.cn',
};

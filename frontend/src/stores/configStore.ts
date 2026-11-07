import { create } from 'zustand';
import {
  config,
  GetConfig,
  SaveConfig,
  GetConnectionStatus,
  StartProxy,
  StopProxy,
  SetProxyMode,
  SetRoutingMode,
} from '../api';
import { useThemeStore } from './themeStore';

interface ConfigState {
  appConfig: config.AppConfig;
  status: config.ConnectionStatus;
  lastError: string;
  loading: boolean;
  loadConfig: () => Promise<void>;
  updateConfig: (partial: Partial<config.AppConfig>) => Promise<void>;
  toggleConnection: () => Promise<void>;
  changeProxyMode: (mode: config.AppConfig['proxyMode']) => Promise<void>;
  changeRoutingMode: (mode: config.AppConfig['routingMode']) => Promise<void>;
  setStatus: (status: config.ConnectionStatus) => void;
}

export const DEFAULT_DIRECT_RULES = [
  'cn',
  'baidu.com',
  'qq.com',
  'weixin.qq.com',
  'taobao.com',
  'tmall.com',
  'jd.com',
  'alicdn.com',
  'aliyun.com',
  'bilibili.com',
  '163.com',
  '126.com',
  'zhihu.com',
  'douyin.com',
  'weibo.com',
  'meituan.com',
  'apple.com',
  'icloud.com',
  '192.168.0.0/16',
  '10.0.0.0/8',
  '172.16.0.0/12',
];

export const DEFAULT_PROXY_RULES = [
  'google.com',
  'googleapis.com',
  'gstatic.com',
  'youtube.com',
  'ytimg.com',
  'ggpht.com',
  'github.com',
  'githubusercontent.com',
  'openai.com',
  'chatgpt.com',
  'anthropic.com',
  'claude.ai',
  'x.com',
  'twitter.com',
  't.me',
  'telegram.org',
  'cloudflare.com',
  'wikipedia.org',
  'docker.com',
  'huggingface.co',
];

export const DEFAULT_BLOCK_RULES = [
  'doubleclick.net',
  'googlesyndication.com',
  'googleadservices.com',
  'adservice.google.com',
  'cpro.baidu.com',
  'pos.baidu.com',
  'tanx.com',
  'miaozhen.com',
  'umeng.com',
];

const defaultCfg: config.AppConfig = {
  socksPort: 10808,
  httpPort: 10809,
  allowLan: false,
  proxyMode: 'system',
  routingMode: 'rule',
  directDns: '223.5.5.5',
  remoteDns: 'https://1.1.1.1/dns-query',
  enableDoh: true,
  themeMode: 'dark',
  themeColor: 'blue',
  closeToTray: false,
  autoStart: false,
  latencyTestUrl: 'http://cp.cloudflare.com/generate_204',
  latencyTimeout: 3000,
  latencyMaxConc: 20,
  activeNodeId: '',
  activeGroupId: 'default',
  coreType: 'xray',
  customDirectRules: DEFAULT_DIRECT_RULES,
  customProxyRules: DEFAULT_PROXY_RULES,
  customBlockRules: DEFAULT_BLOCK_RULES,
};

export const useConfigStore = create<ConfigState>((set, get) => ({
  appConfig: defaultCfg,
  status: 'disconnected',
  lastError: '',
  loading: false,

  loadConfig: async () => {
    const [cfg, st] = await Promise.all([GetConfig(), GetConnectionStatus()]);
    useThemeStore
      .getState()
      .syncFromConfig(cfg.themeMode || 'dark', cfg.themeColor || 'blue');
    set({ appConfig: cfg, status: st });
  },

  updateConfig: async (partial) => {
    const merged = { ...get().appConfig, ...partial };
    set({ appConfig: merged });
    await SaveConfig(merged);
  },

  toggleConnection: async () => {
    const cur = get().status;
    if (cur === 'connecting' || cur === 'disconnecting') return;

    set({ loading: true, lastError: '' });
    try {
      if (cur === 'connected') {
        set({ status: 'disconnecting' });
        await StopProxy();
        set({ status: 'disconnected', loading: false });
      } else {
        set({ status: 'connecting' });
        await StartProxy();
        set({ status: 'connected', loading: false });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      set({ status: 'error', lastError: msg, loading: false });
      throw err;
    }
  },

  changeProxyMode: async (mode) => {
    set((s) => ({ appConfig: { ...s.appConfig, proxyMode: mode } }));
    await SetProxyMode(mode);
  },

  changeRoutingMode: async (mode) => {
    set((s) => ({ appConfig: { ...s.appConfig, routingMode: mode } }));
    await SetRoutingMode(mode);
  },

  setStatus: (status) => set({ status }),
}));

import { create } from 'zustand';
import { config, GetLogs, ClearLogs } from '../api';

export interface TrafficPoint {
  up: number;
  down: number;
}

const HISTORY_SIZE = 60;

function createInitialHistory(): TrafficPoint[] {
  return Array.from({ length: HISTORY_SIZE }, () => ({ up: 0, down: 0 }));
}

interface TrafficState {
  current: config.TrafficStats;
  history: TrafficPoint[];
  logs: string[];
  pushTrafficStats: (stats: config.TrafficStats) => void;
  appendLog: (line: string) => void;
  loadInitialLogs: () => Promise<void>;
  clearAllLogs: () => Promise<void>;
}

export const useTrafficStore = create<TrafficState>((set) => ({
  current: {
    uploadSpeed: 0,
    downloadSpeed: 0,
    totalUpload: 0,
    totalDownload: 0,
    activeConns: 0,
  },
  history: createInitialHistory(),
  logs: [],

  pushTrafficStats: (stats) =>
    set((state) => {
      const nextHistory = [
        ...state.history.slice(1),
        { up: stats.uploadSpeed, down: stats.downloadSpeed },
      ];
      return {
        current: stats,
        history: nextHistory,
      };
    }),

  appendLog: (line) =>
    set((state) => ({
      logs: [...state.logs.slice(-499), line],
    })),

  loadInitialLogs: async () => {
    const logs = await GetLogs();
    set({ logs });
  },

  clearAllLogs: async () => {
    await ClearLogs();
    set({ logs: [] });
  },
}));

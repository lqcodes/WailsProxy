import { create } from 'zustand';
import {
  config,
  GetNodes,
  GetSubscriptions,
  ImportNodesToGroup,
  DeleteNode,
  UpdateNode,
  DeleteInvalidNodes,
  SortNodesByDelay,
  SetActiveNode,
  AddSubscription,
  UpdateSubscription,
  UpdateAllSubscriptions,
  DeleteSubscription,
  MoveNodesToGroup,
  TestNodeDelay,
  TestAllDelay,
  TestGroupDelay,
} from '../api';
import { useConfigStore } from './configStore';

interface NodeState {
  nodes: config.NodeConfig[];
  subscriptions: config.Subscription[];
  selectedGroupId: string;
  searchQuery: string;
  testingDelay: boolean;
  refreshingSub: boolean;

  fetchAll: () => Promise<void>;
  setSelectedGroup: (subId: string) => void;
  setSearchQuery: (q: string) => void;
  selectActiveNode: (nodeId: string) => Promise<void>;
  importShareLinks: (links: string, subId?: string) => Promise<number>;
  updateNode: (node: config.NodeConfig) => Promise<void>;
  removeNode: (id: string) => Promise<void>;
  cleanInvalidNodes: (subId: string) => Promise<number>;
  sortByDelay: () => Promise<void>;
  pingSingleNode: (nodeId: string) => Promise<number>;
  pingGroupNodes: (subId: string) => Promise<void>;
  pingAllNodes: () => Promise<void>;
  applyPingResult: (res: config.PingResultEvent) => void;
  saveSubscription: (sub: Partial<config.Subscription>) => Promise<void>;
  refreshSubscription: (subId: string) => Promise<void>;
  refreshAllSubscriptions: () => Promise<void>;
  removeSubscription: (subId: string) => Promise<void>;
  moveNodesToGroup: (nodeIds: string[], targetSubId: string) => Promise<void>;
}

export const useNodeStore = create<NodeState>((set, get) => ({
  nodes: [],
  subscriptions: [],
  selectedGroupId: 'all',
  searchQuery: '',
  testingDelay: false,
  refreshingSub: false,

  fetchAll: async () => {
    const [nodes, subs] = await Promise.all([GetNodes(), GetSubscriptions()]);
    set({ nodes, subscriptions: subs });
  },

  setSelectedGroup: (subId) => set({ selectedGroupId: subId }),
  setSearchQuery: (q) => set({ searchQuery: q }),

  selectActiveNode: async (nodeId) => {
    await SetActiveNode(nodeId);
    await useConfigStore.getState().loadConfig();
  },

  importShareLinks: async (links, subId) => {
    const targetGroup =
      subId && subId !== 'all'
        ? subId
        : get().selectedGroupId !== 'all'
        ? get().selectedGroupId
        : 'default';
    const added = await ImportNodesToGroup(links, targetGroup);
    await get().fetchAll();
    await useConfigStore.getState().loadConfig();
    return added.length;
  },

  updateNode: async (node) => {
    await UpdateNode(node);
    await get().fetchAll();
    await useConfigStore.getState().loadConfig();
  },

  removeNode: async (id) => {
    await DeleteNode(id);
    await get().fetchAll();
    await useConfigStore.getState().loadConfig();
  },

  cleanInvalidNodes: async (subId) => {
    const removed = await DeleteInvalidNodes(subId);
    await get().fetchAll();
    return removed;
  },

  sortByDelay: async () => {
    await SortNodesByDelay();
    await get().fetchAll();
  },

  pingSingleNode: async (nodeId) => {
    const delay = await TestNodeDelay(nodeId);
    await get().fetchAll();
    return delay;
  },

  pingGroupNodes: async (subId) => {
    set({ testingDelay: true });
    try {
      await TestGroupDelay(subId);
      await get().fetchAll();
    } finally {
      set({ testingDelay: false });
    }
  },

  pingAllNodes: async () => {
    set({ testingDelay: true });
    try {
      await TestAllDelay();
      await get().fetchAll();
    } finally {
      set({ testingDelay: false });
    }
  },

  applyPingResult: (res) => {
    set((state) => ({
      nodes: state.nodes.map((n) =>
        n.id === res.nodeId
          ? { ...n, tcpDelay: res.tcpDelay, realDelay: res.realDelay }
          : n
      ),
    }));
  },

  saveSubscription: async (sub) => {
    await AddSubscription(sub);
    await get().fetchAll();
  },

  refreshSubscription: async (subId) => {
    set({ refreshingSub: true });
    try {
      await UpdateSubscription(subId);
      await get().fetchAll();
    } finally {
      set({ refreshingSub: false });
    }
  },

  refreshAllSubscriptions: async () => {
    set({ refreshingSub: true });
    try {
      await UpdateAllSubscriptions();
      await get().fetchAll();
    } finally {
      set({ refreshingSub: false });
    }
  },

  removeSubscription: async (subId) => {
    await DeleteSubscription(subId);
    if (get().selectedGroupId === subId) {
      set({ selectedGroupId: 'all' });
    }
    await get().fetchAll();
    await useConfigStore.getState().loadConfig();
  },

  moveNodesToGroup: async (nodeIds, targetSubId) => {
    await MoveNodesToGroup(nodeIds, targetSubId);
    await get().fetchAll();
  },
}));

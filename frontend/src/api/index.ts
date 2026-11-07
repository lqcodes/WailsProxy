import * as WailsApp from '../../wailsjs/go/main/App';
import * as WailsRuntime from '../../wailsjs/runtime/runtime';
import { config as WailsModels } from '../../wailsjs/go/models';

export namespace config {
  export type ConnectionStatus =
    | 'disconnected'
    | 'connecting'
    | 'connected'
    | 'disconnecting'
    | 'error';

  export interface AppConfig {
    socksPort: number;
    httpPort: number;
    allowLan: boolean;
    proxyMode: 'system' | 'manual' | 'tun' | string;
    routingMode: 'rule' | 'global' | 'direct' | string;
    directDns: string;
    remoteDns: string;
    enableDoh: boolean;
    themeMode: 'light' | 'dark' | 'auto' | string;
    themeColor: 'blue' | 'green' | 'purple' | 'orange' | 'pink' | string;
    closeToTray: boolean;
    autoStart: boolean;
    latencyTestUrl: string;
    latencyTimeout: number;
    latencyMaxConc: number;
    activeNodeId: string;
    activeGroupId: string;
    coreType: 'sing-box' | 'xray' | string;
    customDirectRules: string[];
    customProxyRules: string[];
    customBlockRules: string[];
  }

  export type NodeConfig = WailsModels.NodeConfig;
  export type Subscription = WailsModels.Subscription;

  export interface TrafficStats {
    uploadSpeed: number;
    downloadSpeed: number;
    totalUpload: number;
    totalDownload: number;
    activeConns: number;
  }

  export interface PingResultEvent {
    nodeId: string;
    tcpDelay: number;
    realDelay: number;
  }

  export interface SubscriptionUpdatedEvent {
    subId: string;
    nodeCount: number;
    error?: string;
  }

  export interface DependencyItem {
    id: string;
    name: string;
    required: boolean;
    installed: boolean;
    version: string;
    path: string;
    description: string;
    installCmd?: string;
    error?: string;
  }

  export interface DependencyCheckResult {
    ready: boolean;
    platform?: string;
    binDir: string;
    items: DependencyItem[];
    message: string;
  }
}

export const StartProxy = () => WailsApp.StartProxy();
export const StopProxy = () => WailsApp.StopProxy();
export const GetConnectionStatus = async (): Promise<config.ConnectionStatus> => {
  const st = await WailsApp.GetConnectionStatus();
  return (st as config.ConnectionStatus) || 'disconnected';
};

export const GetNodes = () => WailsApp.GetNodes();
export const GetNodesByGroup = (subId: string) => WailsApp.GetNodesByGroup(subId);
export const ImportNodes = (links: string) => WailsApp.ImportNodes(links);
export const ImportNodesToGroup = (links: string, subId: string) =>
  WailsApp.ImportNodesToGroup(links, subId);
export const DeleteNode = (id: string) => WailsApp.DeleteNode(id);
export const UpdateNode = (node: config.NodeConfig) =>
  (WailsApp as unknown as {
    UpdateNode: (n: config.NodeConfig) => Promise<void>;
  }).UpdateNode(WailsModels.NodeConfig.createFrom(node));
export const DeleteInvalidNodes = (subId: string) =>
  WailsApp.DeleteInvalidNodes(subId);
export const SortNodesByDelay = () => WailsApp.SortNodesByDelay();
export const SetActiveNode = (nodeId: string) => WailsApp.SetActiveNode(nodeId);

export const GetSubscriptions = () => WailsApp.GetSubscriptions();
export const AddSubscription = (sub: Partial<config.Subscription>) =>
  WailsApp.AddSubscription(WailsModels.Subscription.createFrom(sub));
export const UpdateSubscription = (subId: string) =>
  WailsApp.UpdateSubscription(subId);
export const UpdateAllSubscriptions = () => WailsApp.UpdateAllSubscriptions();
export const DeleteSubscription = (subId: string) =>
  WailsApp.DeleteSubscription(subId);
export const MoveNodesToGroup = (nodeIds: string[], targetSubId: string) =>
  (WailsApp as unknown as {
    MoveNodesToGroup: (nodeIds: string[], targetSubId: string) => Promise<void>;
  }).MoveNodesToGroup(nodeIds, targetSubId);

export const TestNodeDelay = (nodeId: string) => WailsApp.TestNodeDelay(nodeId);
export const TestAllDelay = () => WailsApp.TestAllDelay();
export const TestGroupDelay = (subId: string) => WailsApp.TestGroupDelay(subId);

export const GetConfig = async (): Promise<config.AppConfig> => {
  return WailsApp.GetConfig();
};
export const SaveConfig = (cfg: config.AppConfig) =>
  WailsApp.SaveConfig(WailsModels.AppConfig.createFrom(cfg));
export const SetProxyMode = (mode: string) => WailsApp.SetProxyMode(mode);
export const SetRoutingMode = (mode: string) => WailsApp.SetRoutingMode(mode);

export const GetLogs = () => WailsApp.GetLogs();
export const ClearLogs = () => WailsApp.ClearLogs();
export const GetTerminalProxyCommand = (shell: string) =>
  WailsApp.GetTerminalProxyCommand(shell);

export const CheckDependencies = async (): Promise<config.DependencyCheckResult> => {
  return (await (WailsApp as unknown as {
    CheckDependencies: () => Promise<config.DependencyCheckResult>;
  }).CheckDependencies());
};

export const InstallDependencies = async (): Promise<config.DependencyCheckResult> => {
  return (await (WailsApp as unknown as {
    InstallDependencies: () => Promise<config.DependencyCheckResult>;
  }).InstallDependencies());
};

export const EventsOn = WailsRuntime.EventsOn;
export const EventsOff = WailsRuntime.EventsOff;
export const EventsEmit = WailsRuntime.EventsEmit;
export const WindowShow = WailsRuntime.WindowShow;
export const WindowHide = WailsRuntime.WindowHide;
export const WindowMinimise = WailsRuntime.WindowMinimise;
export const WindowToggleMaximise = WailsRuntime.WindowToggleMaximise;
export const Quit = WailsRuntime.Quit;
export const ClipboardSetText = WailsRuntime.ClipboardSetText;

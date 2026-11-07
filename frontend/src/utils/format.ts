// 流量速率格式化：自动换算为 B/s, KB/s, MB/s, GB/s
export function formatSpeed(bytesPerSec: number): string {
  if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
  let val = bytesPerSec;
  let idx = 0;
  while (val >= 1024 && idx < units.length - 1) {
    val /= 1024;
    idx++;
  }
  return `${val >= 100 ? val.toFixed(0) : val.toFixed(1)} ${units[idx]}`;
}

// 累计流量格式化：自动换算为 B, KB, MB, GB, TB
export function formatTraffic(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let val = bytes;
  let idx = 0;
  while (val >= 1024 && idx < units.length - 1) {
    val /= 1024;
    idx++;
  }
  return `${val >= 100 ? val.toFixed(1) : val.toFixed(2)} ${units[idx]}`;
}

// Unix 时间戳格式化
export function formatTimestamp(ts: number): string {
  if (!ts || ts <= 0) return '从未更新';
  const d = new Date(ts * 1000);
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// 协议标签颜色映射
export function getProtocolTagColor(protocol: string): string {
  switch ((protocol || '').toLowerCase()) {
    case 'vless':
      return 'geekblue';
    case 'vmess':
      return 'purple';
    case 'trojan':
      return 'magenta';
    case 'ss':
    case 'shadowsocks':
      return 'cyan';
    case 'hysteria2':
    case 'hy2':
      return 'orange';
    case 'tuic':
      return 'green';
    default:
      return 'blue';
  }
}

// 延迟数值颜色映射
export function getDelayStatusColor(delay: number): string {
  if (delay < 0) return 'error';
  if (delay === 0) return 'default';
  if (delay < 100) return 'success';
  if (delay < 260) return 'warning';
  return 'volcano';
}

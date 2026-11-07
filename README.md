# WailsProxy - 现代跨平台代理客户端

基于 **Go + Wails v2 + React 18 + TypeScript + Ant Design v5 + Zustand** 构建的现代化跨平台代理客户端（支持 **macOS [Apple Silicon 苹果芯 / Intel 英特尔芯]**、**Windows**、**Linux**）。

---

## 一、三大操作系统构建 App 与安装指南 (macOS / Windows / Linux)

本项目提供了一键跨平台构建脚本 [`scripts/build-all.sh`](file:///Users/peter/AntigravityProjects/WailsProxy/scripts/build-all.sh)，也可直接按以下各平台命令构建并安装：

```bash
# 使用一键脚本构建指定平台：
./scripts/build-all.sh mac-arm64       # macOS 苹果芯 (M1/M2/M3/M4)
./scripts/build-all.sh mac-intel       # macOS 英特尔芯 (Intel x86_64)
./scripts/build-all.sh mac-universal   # macOS 双架构通用包 (苹果芯 + 英特尔芯)
./scripts/build-all.sh windows-amd64   # Windows 64位 (.exe)
./scripts/build-all.sh linux-amd64     # Linux 64位
```

---

### 1. macOS 构建与安装（区分苹果芯 `arm64` 与英特尔芯 `amd64`）

#### 1.1 前置环境安装 (macOS)
```bash
# 安装 Go、Node.js 与 Wails CLI
brew install go node
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### 1.2 构建应用包 (`WailsProxy.app`)

##### 方式 A：苹果芯 Mac (Apple Silicon M1 / M2 / M3 / M4 — `darwin/arm64`)
```bash
# 1. 编译 Apple Silicon 原生应用包
wails build -platform darwin/arm64 -clean

# 2. 将内核依赖 (xray / sing-box / geoip.dat / geosite.dat) 内置打包进 .app 内部
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
cp -f build/bin/WailsProxy.app/Contents/MacOS/WailsProxy build/bin/WailsProxy
```

##### 方式 B：英特尔芯 Mac (Intel x86_64 — `darwin/amd64`)
```bash
# 1. 编译 Intel x86_64 应用包
wails build -platform darwin/amd64 -clean

# 2. 将 Intel 架构内核依赖内置打包进 .app 内部
# (注：若在 Apple Silicon 机器交叉编译给 Intel Mac，请将 bin/ 下替换为 darwin-amd64 架构的 xray 与 sing-box)
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
```

##### 方式 C：macOS 双架构通用包 (Universal Binary：同时兼容苹果芯 + 英特尔芯)
```bash
wails build -platform darwin/universal -clean
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
```

#### 1.3 macOS 安装与启动命令
```bash
# 安装到 macOS「应用程序 (/Applications)」目录并移除隔离属性
cp -R build/bin/WailsProxy.app /Applications/
xattr -cr /Applications/WailsProxy.app

# 启动应用
open -a /Applications/WailsProxy.app
```

---

### 2. Windows 构建与安装 (`windows/amd64` & `windows/arm64`)

#### 2.1 前置环境安装 (Windows PowerShell)
```powershell
# 使用 winget 安装 Go、Node.js 与 NSIS 安装包制作工具
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
winget install NSIS.NSIS
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### 2.2 构建 Windows 应用与 NSIS 安装程序
```powershell
# 1. 构建 Windows 64位应用 (.exe) 与 NSIS 安装包 (-nsis)
wails build -platform windows/amd64 -nsis -clean

# (可选) 构建 Windows ARM64 版本
wails build -platform windows/arm64 -clean

# 2. 下载并内置 Windows 版内核依赖到发布目录 build\bin\bin\
New-Item -ItemType Directory -Force -Path "build\bin\bin"
Invoke-WebRequest -Uri "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-windows-64.zip" -OutFile "xray.zip"
Expand-Archive -Path "xray.zip" -DestinationPath "build\bin\bin" -Force
Remove-Item "xray.zip"

Invoke-WebRequest -Uri "https://github.com/SagerNet/sing-box/releases/download/v1.11.4/sing-box-1.11.4-windows-amd64.zip" -OutFile "sb.zip"
Expand-Archive -Path "sb.zip" -DestinationPath "sb_tmp" -Force
Get-ChildItem -Path "sb_tmp" -Recurse -Filter "sing-box.exe" | Copy-Item -Destination "build\bin\bin\sing-box.exe" -Force
Remove-Item "sb.zip", "sb_tmp" -Recurse -Force
```

#### 2.3 Windows 安装与运行命令
```powershell
# 方式 1：运行 NSIS 安装向导安装到系统
Start-Process ".\build\bin\WailsProxy-amd64-installer.exe"

# 方式 2：直接运行免安装便携版
Start-Process ".\build\bin\WailsProxy.exe"
```

---

### 3. Linux 构建与安装 (`linux/amd64` & `linux/arm64`)

#### 3.1 前置系统依赖安装 (Ubuntu / Debian / Fedora / Arch)
```bash
# Ubuntu / Debian (安装 GTK3 与 WebKit2GTK 开发库)
sudo apt update && sudo apt install -y build-essential libgtk-3-dev libwebkit2gtk-4.0-dev curl unzip tar

# Fedora / RHEL
sudo dnf install -y gtk3-devel webkit2gtk3-devel gcc-c++ curl unzip tar

# Arch Linux
sudo pacman -S --needed base-devel gtk3 webkit2gtk curl unzip tar
```

#### 3.2 构建 Linux 可执行文件并内置内核依赖
```bash
# 1. 编译 Linux x86_64 (amd64) 版本（若为 ARM64 架构则改为 linux/arm64）
wails build -platform linux/amd64 -clean

# 2. 下载并打包 Linux 架构的 xray、sing-box 与 geo 规则文件到 build/bin/bin/
mkdir -p build/bin/bin
curl -L -o /tmp/xray-linux.zip https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip
unzip -o /tmp/xray-linux.zip xray geoip.dat geosite.dat -d build/bin/bin/
chmod +x build/bin/bin/xray

curl -L -o /tmp/sb-linux.tar.gz https://github.com/SagerNet/sing-box/releases/download/v1.11.4/sing-box-1.11.4-linux-amd64.tar.gz
tar -xzf /tmp/sb-linux.tar.gz -C /tmp
find /tmp -name "sing-box" -type f -exec cp -f {} build/bin/bin/sing-box \;
chmod +x build/bin/bin/sing-box
rm -f /tmp/xray-linux.zip /tmp/sb-linux.tar.gz
```

#### 3.3 Linux 系统级安装与桌面快捷方式创建
```bash
# 1. 安装到 /opt/WailsProxy 并创建全局命令软链
sudo mkdir -p /opt/WailsProxy/bin
sudo cp -f build/bin/WailsProxy /opt/WailsProxy/
sudo cp -rf build/bin/bin/* /opt/WailsProxy/bin/
sudo cp -f build/appicon.png /opt/WailsProxy/appicon.png
sudo chmod +x /opt/WailsProxy/WailsProxy /opt/WailsProxy/bin/xray /opt/WailsProxy/bin/sing-box
sudo ln -sf /opt/WailsProxy/WailsProxy /usr/local/bin/wailsproxy

# 2. 创建 Linux 桌面应用图标 (/usr/share/applications/wailsproxy.desktop)
cat << 'EOF' | sudo tee /usr/share/applications/wailsproxy.desktop
[Desktop Entry]
Name=WailsProxy
Comment=Modern Cross-Platform Proxy Client
Exec=/opt/WailsProxy/WailsProxy
Icon=/opt/WailsProxy/appicon.png
Terminal=false
Type=Application
Categories=Network;Utility;
EOF
```

---

## 二、开发调试与全量测试 (Dev & Testing)

### 1. Wails 开发热重载模式
```bash
wails dev
```

### 2. 运行全量单元与端到端测试
```bash
go test -v -race -cover -count=1 ./...
```

---

## 三、核心特性概览

1. **内核自动托管与开箱即用**：
   - 根据不同操作系统自动设置并创建默认运行目录（macOS `~/Library/Application Support/WailsProxy`、Windows `%AppData%\WailsProxy`、Linux `~/.config/WailsProxy`），自动同步内置 **Xray-core**、**Sing-box** 与 **GeoIP/GeoSite** 规则库。
   - 启动时后台自动检测依赖，缺失项支持在弹窗内单点「点击安装」自动下载部署，已安装且可加载项显示绿色打勾「已就绪」，无需用户手动配置任何路径。
2. **防 DNS 泄漏与智能分流**：
   - 内置防 DNS 泄漏引擎（非中国大陆域名强制通过代理节点走远程 DoH/TCP DNS 解析，杜绝本地 ISP 抢答污染）。
   - 默认预置广告拦截、局域网直连、国内直连、AI 平台与流媒体/学术开发平台代理规则。
3. **全协议支持与通用节点编辑器**：
   - 支持 `VLESS` (含 Reality / XTLS-Vision / 后量子 `pqv`)、`VMess`、`Trojan`、`Shadowsocks`、`Hysteria2`、`TUIC` 协议及 Base64 / Clash YAML / Sing-box JSON 订阅。
   - 节点表格支持拖拽列宽、溢出自动隐藏、点击任意列弹出多协议自适应编辑弹窗。
4. **多主题无缝切换**：
   - 支持 **浅色 (Light)**、**暗黑 (Dark)**、**跟随系统 (Auto)** 与 5 款主题色（极客蓝、极光绿、科技紫、晚霞橙、赛博朋克），主界面与全部弹窗实时同步适配。

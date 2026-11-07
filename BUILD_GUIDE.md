# WailsProxy 三大操作系统 (macOS / Windows / Linux) 构建与安装指南

本文档提供 **macOS（区分 Apple Silicon 苹果芯 与 Intel 英特尔芯）**、**Windows** 与 **Linux** 三大操作系统的完整构建打包、内核依赖内置与系统安装命令。

---

## 一、macOS 构建与安装（区分苹果芯 `arm64` 与英特尔芯 `amd64`）

### 1.1 前置环境安装 (macOS)
```bash
# 安装 Go、Node.js 与 Wails CLI
brew install go node
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 1.2 构建应用包 (`WailsProxy.app`)

#### 方式 A：苹果芯 Mac (Apple Silicon M1 / M2 / M3 / M4 — `darwin/arm64`)
```bash
# 1. 编译 Apple Silicon 原生应用包
wails build -platform darwin/arm64 -clean

# 2. 将内核依赖 (xray / sing-box / geoip.dat / geosite.dat) 打包进 .app 内部
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
cp -f build/bin/WailsProxy.app/Contents/MacOS/WailsProxy build/bin/WailsProxy
```

#### 方式 B：英特尔芯 Mac (Intel x86_64 — `darwin/amd64`)
```bash
# 1. 下载 Intel 架构对应的 xray / sing-box 二进制 (若在构建机打包给 Intel Mac)
# Xray Intel: https://github.com/XTLS/Xray-core/releases/latest/download/Xray-macos-64.zip
# Sing-box Intel: https://github.com/SagerNet/sing-box/releases/download/v1.11.4/sing-box-1.11.4-darwin-amd64.tar.gz

# 2. 编译 Intel x86_64 应用包
wails build -platform darwin/amd64 -clean

# 3. 将内核依赖打包进 .app 内部
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
```

#### 方式 C：macOS 双架构通用包 (Universal Binary：同时兼容苹果芯 + 英特尔芯)
```bash
wails build -platform darwin/universal -clean
mkdir -p build/bin/WailsProxy.app/Contents/MacOS/bin build/bin/WailsProxy.app/Contents/Resources/bin
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/MacOS/bin/
cp -f bin/xray bin/sing-box bin/geoip.dat bin/geosite.dat build/bin/WailsProxy.app/Contents/Resources/bin/
```

### 1.3 macOS 安装与启动命令
```bash
# 安装到 macOS「应用程序 (/Applications)」目录并移除隔离属性
cp -R build/bin/WailsProxy.app /Applications/
xattr -cr /Applications/WailsProxy.app

# 启动应用
open -a /Applications/WailsProxy.app
```

---

## 二、Windows 构建与安装 (`windows/amd64` & `windows/arm64`)

### 2.1 前置环境安装 (Windows PowerShell)
```powershell
# 使用 winget 安装 Go、Node.js 与 NSIS 安装包制作工具
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
winget install NSIS.NSIS
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 2.2 构建 Windows 应用与 NSIS 安装程序
```powershell
# 1. 构建 Windows 64位绿色版 (.exe) 与 NSIS 安装包 (-nsis)
wails build -platform windows/amd64 -nsis -clean

# (可选) 构建 Windows ARM64 版本
wails build -platform windows/arm64 -clean

# 2. 将 Windows 版内核依赖放入发布目录 bin\
New-Item -ItemType Directory -Force -Path "build\bin\bin"
Invoke-WebRequest -Uri "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-windows-64.zip" -OutFile "xray.zip"
Expand-Archive -Path "xray.zip" -DestinationPath "build\bin\bin" -Force
Remove-Item "xray.zip"

Invoke-WebRequest -Uri "https://github.com/SagerNet/sing-box/releases/download/v1.11.4/sing-box-1.11.4-windows-amd64.zip" -OutFile "sb.zip"
Expand-Archive -Path "sb.zip" -DestinationPath "sb_tmp" -Force
Get-ChildItem -Path "sb_tmp" -Recurse -Filter "sing-box.exe" | Copy-Item -Destination "build\bin\bin\sing-box.exe" -Force
Remove-Item "sb.zip", "sb_tmp" -Recurse -Force
```

### 2.3 Windows 安装与运行命令
```powershell
# 方式 1：直接运行 NSIS 安装向导
Start-Process ".\build\bin\WailsProxy-amd64-installer.exe"

# 方式 2：直接运行绿色便携版
Start-Process ".\build\bin\WailsProxy.exe"
```

---

## 三、Linux 构建与安装 (`linux/amd64` & `linux/arm64`)

### 3.1 前置系统依赖安装 (Ubuntu / Debian / Fedora / Arch)
```bash
# Ubuntu / Debian (安装 GTK3 與 WebKit2GTK 开发库)
sudo apt update && sudo apt install -y build-essential libgtk-3-dev libwebkit2gtk-4.0-dev curl unzip tar

# Fedora / RHEL
sudo dnf install -y gtk3-devel webkit2gtk3-devel gcc-c++ curl unzip tar

# Arch Linux
sudo pacman -S --needed base-devel gtk3 webkit2gtk curl unzip tar
```

### 3.2 构建 Linux 可执行文件并内置内核依赖
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

### 3.3 Linux 系统级安装与桌面快捷方式创建
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

package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"wailsproxy/internal/config"
)

func platformLabel() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "darwin/arm64 (macOS Apple Silicon 苹果芯)"
		}
		return fmt.Sprintf("darwin/%s (macOS Intel 英特尔芯)", runtime.GOARCH)
	case "windows":
		return fmt.Sprintf("windows/%s (Windows)", runtime.GOARCH)
	case "linux":
		return fmt.Sprintf("linux/%s (Linux)", runtime.GOARCH)
	default:
		return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func platformInstallCmd(id string) string {
	switch runtime.GOOS {
	case "darwin":
		switch id {
		case "xray":
			return "brew install xray"
		case "sing-box":
			return "brew install sing-box"
		case "geodata":
			return "curl -L -o bin/geoip.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat && curl -L -o bin/geosite.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
		default:
			return "networksetup -listallnetworkservices"
		}
	case "windows":
		switch id {
		case "xray":
			return "winget install --id XTLS.Xray --silent  (或 scoop install xray)"
		case "sing-box":
			return "winget install --id SagerNet.sing-box --silent  (或 scoop install sing-box)"
		case "geodata":
			return "powershell -Command \"Invoke-WebRequest -Uri 'https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat' -OutFile 'bin\\geoip.dat'; Invoke-WebRequest -Uri 'https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat' -OutFile 'bin\\geosite.dat'\""
		default:
			return "reg query \"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\""
		}
	default: // linux
		switch id {
		case "xray":
			return "bash -c \"$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)\" @ install"
		case "sing-box":
			return "curl -fsSL https://sing-box.app/install.sh | sh"
		case "geodata":
			return "curl -L -o bin/geoip.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat && curl -L -o bin/geosite.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
		default:
			return "gsettings list-schemas | grep system.proxy"
		}
	}
}

// syncBundledAssetsIfPresent 若应用安装包内置了内核与规则库，自动同步到当前系统默认 binDir（免用户手动配置路径）
func (m *Manager) syncBundledAssetsIfPresent() {
	_ = os.MkdirAll(m.binDir, 0755)
	_ = os.MkdirAll(m.dataDir, 0755)

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execDir := filepath.Dir(execPath)
	bundleDirs := []string{
		filepath.Join(execDir, "bin"),
		filepath.Join(execDir, "..", "Resources", "bin"),
	}

	assets := []struct {
		name string
		perm os.FileMode
	}{
		{binaryName("xray"), 0755},
		{binaryName("sing-box"), 0755},
		{"geoip.dat", 0644},
		{"geosite.dat", 0644},
	}

	for _, asset := range assets {
		dst := filepath.Join(m.binDir, asset.name)
		if fileExists(dst) {
			continue
		}
		for _, dir := range bundleDirs {
			src := filepath.Join(dir, asset.name)
			if fileExists(src) {
				_ = copyFile(src, dst, asset.perm)
				break
			}
		}
	}
}

// CheckDependencies 后台检测开源内核二进制、Geo 规则库与系统代理工具是否全部安装且可正常加载（不暴露本地磁盘路径）
func (m *Manager) CheckDependencies() config.DependencyCheckResult {
	m.syncBundledAssetsIfPresent()

	var items []config.DependencyItem
	allReady := true

	// 1. 检测 Xray-core 内核（验证已安装且可正常加载运行）
	xrayBin := m.FindCoreBinary("xray")
	xrayItem := config.DependencyItem{
		Id:          "xray",
		Name:        "Xray-core 代理引擎",
		Required:    true,
		Description: "支持 VLESS Reality (含后量子 ML-DSA-65 pqv)、VMess、Trojan、Shadowsocks 等核心协议",
	}
	if xrayBin != "" {
		if ver, ok := verifyBinaryLoadable(xrayBin, "version"); ok {
			xrayItem.Installed = true
			xrayItem.Version = ver
		} else {
			xrayItem.Installed = false
			xrayItem.Error = "内核文件无法加载执行，请点击安装重新部署"
			allReady = false
		}
	} else {
		xrayItem.Installed = false
		xrayItem.Error = "尚未安装 Xray-core 引擎，请点击右侧按钮自动安装"
		allReady = false
	}
	items = append(items, xrayItem)

	// 2. 检测 Sing-box 内核（验证已安装且可正常加载运行）
	sbBin := m.FindCoreBinary("sing-box")
	sbItem := config.DependencyItem{
		Id:          "sing-box",
		Name:        "Sing-box 通用代理引擎",
		Required:    true,
		Description: "支持 Hysteria2、TUIC、VLESS、TUN 透明虚拟网卡与高性能分流引擎",
	}
	if sbBin != "" {
		if ver, ok := verifyBinaryLoadable(sbBin, "version"); ok {
			sbItem.Installed = true
			sbItem.Version = ver
		} else {
			sbItem.Installed = false
			sbItem.Error = "内核文件无法加载执行，请点击安装重新部署"
			allReady = false
		}
	} else {
		sbItem.Installed = false
		sbItem.Error = "尚未安装 Sing-box 引擎，请点击右侧按钮自动安装"
		allReady = false
	}
	items = append(items, sbItem)

	// 3. 检测 GeoIP / GeoSite 路由规则数据
	geoipPath := filepath.Join(m.binDir, "geoip.dat")
	geositePath := filepath.Join(m.binDir, "geosite.dat")
	geoItem := config.DependencyItem{
		Id:          "geodata",
		Name:        "GeoIP & GeoSite 路由分流数据库",
		Required:    true,
		Description: "用于规则模式 (Rule Mode) 下国内外域名与 IP 智能分流及广告拦截",
	}
	if fileExists(geoipPath) && fileExists(geositePath) {
		geoItem.Installed = true
		geoItem.Version = "geoip.dat + geosite.dat"
	} else {
		geoItem.Installed = false
		geoItem.Error = "缺少路由分流数据库，请点击右侧按钮自动安装"
		allReady = false
	}
	items = append(items, geoItem)

	// 4. 检测系统代理控制服务
	sysItem := config.DependencyItem{
		Id:          "sysproxy",
		Name:        "系统网络代理控制服务",
		Required:    true,
		Description: "用于自动配置与安全还原操作系统全局 HTTP / HTTPS / SOCKS5 代理",
	}
	sysCmd := "networksetup"
	switch runtime.GOOS {
	case "windows":
		sysCmd = "reg"
	case "linux":
		sysCmd = "gsettings"
	}
	if _, err := exec.LookPath(sysCmd); err == nil {
		sysItem.Installed = true
		sysItem.Version = fmt.Sprintf("%s (%s)", sysCmd, runtime.GOOS)
	} else {
		sysItem.Installed = true
		sysItem.Version = runtime.GOOS
	}
	items = append(items, sysItem)

	msg := "所有核心依赖组件均已安装并可正常加载，添加订阅即可立即连接使用。"
	if !allReady {
		msg = "检测到存在未安装或无法加载的核心组件，请点击对应项右侧的「点击安装」或底部「一键全部安装」自动完成配置。"
	}

	return config.DependencyCheckResult{
		Ready:    allReady,
		Platform: platformLabel(),
		BinDir:   "",
		Items:    items,
		Message:  msg,
	}
}

// InstallDependencies 跨平台（macOS / Windows / Linux）一键搜寻、调用包管理器或从官方源下载解压安装缺失依赖
func (m *Manager) InstallDependencies() (config.DependencyCheckResult, error) {
	if err := os.MkdirAll(m.binDir, 0755); err != nil {
		return m.CheckDependencies(), fmt.Errorf("创建 bin 目录失败: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	cwd, _ := os.Getwd()

	candidateRoots := []string{
		filepath.Join(cwd, "bin"),
		filepath.Join(execDir, "bin"),
		filepath.Join(execDir, "..", "Resources", "bin"),
		filepath.Join(execDir, "..", "..", "..", "..", "bin"),
		filepath.Join(homeDir, "Library", "Application Support", "v2rayN", "bin"),
		"/Applications/v2rayN.app/Contents/MacOS/bin",
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		filepath.Join(homeDir, "scoop", "shims"),
		filepath.Join(homeDir, "AppData", "Local", "Microsoft", "WinGet", "Links"),
	}

	// 阶段 1：从应用包内内置资源或系统本地已安装目录复制
	targetXray := filepath.Join(m.binDir, binaryName("xray"))
	if !fileExists(targetXray) {
		for _, root := range candidateRoots {
			for _, rel := range []string{binaryName("xray"), filepath.Join("xray", binaryName("xray"))} {
				src := filepath.Join(root, rel)
				if fileExists(src) {
					_ = copyFile(src, targetXray, 0755)
					break
				}
			}
			if fileExists(targetXray) {
				break
			}
		}
	}

	targetSB := filepath.Join(m.binDir, binaryName("sing-box"))
	if !fileExists(targetSB) {
		for _, root := range candidateRoots {
			for _, rel := range []string{binaryName("sing-box"), filepath.Join("sing_box", binaryName("sing-box")), filepath.Join("sing-box", binaryName("sing-box"))} {
				src := filepath.Join(root, rel)
				if fileExists(src) {
					_ = copyFile(src, targetSB, 0755)
					break
				}
			}
			if fileExists(targetSB) {
				break
			}
		}
	}

	for _, datName := range []string{"geoip.dat", "geosite.dat"} {
		targetDat := filepath.Join(m.binDir, datName)
		if !fileExists(targetDat) {
			for _, root := range candidateRoots {
				for _, rel := range []string{datName, filepath.Join("xray", datName)} {
					src := filepath.Join(root, rel)
					if fileExists(src) {
						_ = copyFile(src, targetDat, 0644)
						break
					}
				}
				if fileExists(targetDat) {
					break
				}
			}
		}
	}

	// 阶段 2：若仍缺失，尝试通过平台包管理器 (macOS brew / Windows winget / Linux apt) 或官方 Release 自动下载解压
	if !fileExists(targetXray) {
		m.trySystemPackageInstall("xray", targetXray)
	}
	if !fileExists(targetXray) {
		_ = m.downloadXrayRelease(targetXray)
	}

	if !fileExists(targetSB) {
		m.trySystemPackageInstall("sing-box", targetSB)
	}
	if !fileExists(targetSB) {
		_ = m.downloadSingBoxRelease(targetSB)
	}

	for _, datName := range []string{"geoip.dat", "geosite.dat"} {
		targetDat := filepath.Join(m.binDir, datName)
		if !fileExists(targetDat) {
			url := "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/" + datName
			_ = downloadFileTo(url, targetDat, 0644, 45*time.Second)
		}
	}

	res := m.CheckDependencies()
	if !res.Ready {
		return res, fmt.Errorf("仍有依赖项未就绪: %s", res.Message)
	}
	return res, nil
}

func (m *Manager) trySystemPackageInstall(pkg string, targetPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "darwin":
		if brewPath, err := exec.LookPath("brew"); err == nil {
			_ = exec.CommandContext(ctx, brewPath, "install", pkg).Run()
		}
	case "windows":
		if scoopPath, err := exec.LookPath("scoop"); err == nil {
			_ = exec.CommandContext(ctx, scoopPath, "install", pkg).Run()
		} else if wingetPath, err := exec.LookPath("winget"); err == nil {
			winId := "XTLS.Xray"
			if pkg == "sing-box" {
				winId = "SagerNet.sing-box"
			}
			_ = exec.CommandContext(ctx, wingetPath, "install", "--id", winId, "--silent", "--accept-package-agreements", "--accept-source-agreements").Run()
		}
	}

	if found, err := exec.LookPath(binaryName(pkg)); err == nil && fileExists(found) {
		_ = copyFile(found, targetPath, 0755)
	}
}

func (m *Manager) downloadXrayRelease(targetXray string) error {
	var asset string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			asset = "Xray-macos-arm64-v8a.zip"
		} else {
			asset = "Xray-macos-64.zip"
		}
	case "windows":
		if runtime.GOARCH == "arm64" {
			asset = "Xray-windows-arm64-v8a.zip"
		} else {
			asset = "Xray-windows-64.zip"
		}
	default: // linux
		if runtime.GOARCH == "arm64" {
			asset = "Xray-linux-arm64-v8a.zip"
		} else {
			asset = "Xray-linux-64.zip"
		}
	}

	dlURL := "https://github.com/XTLS/Xray-core/releases/latest/download/" + asset
	tmpZip := filepath.Join(m.binDir, "xray-dl.zip")
	defer os.Remove(tmpZip)

	if err := downloadFileTo(dlURL, tmpZip, 0644, 90*time.Second); err != nil {
		return err
	}
	return extractZipTargets(tmpZip, m.binDir, map[string]os.FileMode{
		binaryName("xray"): 0755,
		"geoip.dat":        0644,
		"geosite.dat":      0644,
	})
}

func (m *Manager) downloadSingBoxRelease(targetSB string) error {
	ver := "1.11.4"
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	switch runtime.GOOS {
	case "windows":
		asset := fmt.Sprintf("sing-box-%s-windows-%s.zip", ver, arch)
		dlURL := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/%s", ver, asset)
		tmpZip := filepath.Join(m.binDir, "sb-dl.zip")
		defer os.Remove(tmpZip)
		if err := downloadFileTo(dlURL, tmpZip, 0644, 90*time.Second); err != nil {
			return err
		}
		return extractZipTargets(tmpZip, m.binDir, map[string]os.FileMode{
			binaryName("sing-box"): 0755,
		})
	default: // darwin / linux
		asset := fmt.Sprintf("sing-box-%s-%s-%s.tar.gz", ver, runtime.GOOS, arch)
		dlURL := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/%s", ver, asset)
		tmpTar := filepath.Join(m.binDir, "sb-dl.tar.gz")
		defer os.Remove(tmpTar)
		if err := downloadFileTo(dlURL, tmpTar, 0644, 90*time.Second); err != nil {
			return err
		}
		return extractTarGzBinary(tmpTar, binaryName("sing-box"), targetSB)
	}
}

func downloadFileTo(url string, dst string, perm os.FileMode, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WailsProxy/2.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	_ = out.Close()
	_ = os.Chmod(tmp, perm)
	return os.Rename(tmp, dst)
}

func extractZipTargets(zipPath string, destDir string, targets map[string]os.FileMode) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		perm, want := targets[base]
		if !want || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		dst := filepath.Join(destDir, base)
		tmp := dst + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
		if err == nil {
			_, _ = io.Copy(out, rc)
			_ = out.Close()
			_ = os.Chmod(tmp, perm)
			_ = os.Rename(tmp, dst)
		}
		_ = rc.Close()
	}
	return nil
}

func extractTarGzBinary(tarGzPath string, targetBase string, dstPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == targetBase {
			tmp := dstPath + ".tmp"
			out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				_ = os.Remove(tmp)
				return err
			}
			_ = out.Close()
			_ = os.Chmod(tmp, 0755)
			return os.Rename(tmp, dstPath)
		}
	}
	return fmt.Errorf("未在压缩包中找到 %s", targetBase)
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func copyFile(src, dst string, perm os.FileMode) error {
	if filepath.Clean(src) == filepath.Clean(dst) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	_ = out.Close()
	if err := os.Chmod(tmp, perm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func verifyBinaryLoadable(binPath string, arg string) (string, bool) {
	info, err := os.Stat(binPath)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return "", false
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		_ = os.Chmod(binPath, 0755)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binPath, arg).CombinedOutput()
	if err != nil {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0]), true
	}
	return "已加载", true
}

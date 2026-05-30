package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// backendHealthURL 根据 backend.json 中的信息构建 /health 检测 URL。
// 如果 backend.json 不存在或内容无效，使用默认地址 127.0.0.1:8080。
func backendHealthURL() string {
	if info := ReadBackendInfo(); info != nil {
		return fmt.Sprintf("http://%s:%d/health", info.Host, info.Port)
	}
	return "http://127.0.0.1:8080/health"
}

// isBackendRunning 检测后端服务是否在运行（GET /health）。
func isBackendRunning() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(backendHealthURL())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startBackendProcess 启动后端作为独立子进程。
// 使用当前可执行文件路径 + "backend" 子命令，进程完全脱离父进程。
func startBackendProcess() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	cmd := exec.Command(exePath, "backend")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// 使后端进程独立于代理进程，不受父进程信号影响
	if runtime.GOOS == "windows" {
		// Windows: 创建新进程组，脱离父进程控制台
		cmd.SysProcAttr = syscallSysProcAttr()
	} else {
		// Unix: 创建新进程组
		cmd.SysProcAttr = unixSysProcAttr()
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	logger.Info("autostart", "backend process started: pid=%d", cmd.Process.Pid)
	// 释放子进程，让其独立运行，父进程不等待
	cmd.Process.Release()
	return nil
}

// waitForBackend 轮询 /health 端点，等待后端就绪。
// timeout 为最长等待时间，interval 为轮询间隔。
// 返回 true 表示后端已就绪，false 表示超时。
func waitForBackend(timeout, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		resp, err := client.Get(backendHealthURL())
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(interval)
	}
	return false
}

// ensureBackend 检测后端是否在运行，如果未运行则自动启动。
// 启动失败不阻塞代理，仅记录警告。
func ensureBackend() {
	// 步骤 1：检查后端是否已在运行
	if isBackendRunning() {
		logger.Info("autostart", "backend already running")
		return
	}

	// 步骤 2：后端未运行，启动新进程
	logger.Info("autostart", "backend not running, starting...")
	if err := startBackendProcess(); err != nil {
		logger.Warn("autostart", "failed to start backend: %v", err)
		return
	}

	// 步骤 3：等待后端就绪（最多 5 秒）
	if waitForBackend(5*time.Second, 200*time.Millisecond) {
		info := ReadBackendInfo()
		if info != nil {
			fmt.Printf("   Backend started at http://%s:%d\n", info.Host, info.Port)
		}
		logger.Info("autostart", "backend ready")
	} else {
		logger.Warn("autostart", "backend did not become healthy within 5s")
	}
}

// backendAPIBase 返回后端 API 的基础 URL（从 backend.json 读取）。
func backendAPIBase() string {
	if info := ReadBackendInfo(); info != nil {
		return fmt.Sprintf("http://%s:%d", info.Host, info.Port)
	}
	return "http://127.0.0.1:8080"
}

// registerProxyWithBackend 向后端注册本代理进程。
func registerProxyWithBackend(proxyURL string) {
	pid := fmt.Sprintf("%d", os.Getpid())
	body, _ := json.Marshal(map[string]string{"pid": pid, "proxy_url": proxyURL})
	resp, err := http.Post(backendAPIBase()+"/api/proxy/register", "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Warn("autostart", "proxy register failed: %v", err)
		return
	}
	defer resp.Body.Close()
	logger.Info("autostart", "proxy registered: pid=%s url=%s", pid, proxyURL)
}

// unregisterProxyFromBackend 从后端注销本代理进程。
func unregisterProxyFromBackend() {
	pid := fmt.Sprintf("%d", os.Getpid())
	body, _ := json.Marshal(map[string]string{"pid": pid})
	resp, err := http.Post(backendAPIBase()+"/api/proxy/unregister", "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Warn("autostart", "proxy unregister failed: %v", err)
		return
	}
	defer resp.Body.Close()
	logger.Info("autostart", "proxy unregistered: pid=%s", pid)
}

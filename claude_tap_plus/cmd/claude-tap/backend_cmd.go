package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// runBackend 启动后端服务器模式。
//
// 支持的命令行参数：
//   --port, -p PORT    指定服务器监听端口（默认随机）
//   --db, -d PATH      指定数据库文件路径
//   --host HOST        指定服务器绑定地址
func runBackend(args []string) {
	// 使用默认配置初始化后端服务器配置
	cfg := backend.DefaultConfig()

	// 遍历命令行参数，解析并覆盖默认配置
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		// 解析 --port 或 -p 参数（空格分隔形式，如 --port 8080）
		case (arg == "--port" || arg == "-p") && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &cfg.Port)
		// 解析 --port=PORT 形式（等号连接，如 --port=8080）
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(arg[len("--port="):], "%d", &cfg.Port)
		// 解析 --db 或 -d 参数（空格分隔形式）
		case (arg == "--db" || arg == "-d") && i+1 < len(args):
			i++
			cfg.DBPath = args[i]
		// 解析 --db=PATH 形式
		case strings.HasPrefix(arg, "--db="):
			cfg.DBPath = arg[len("--db="):]
		// 解析 --host 参数（空格分隔形式）
		case arg == "--host" && i+1 < len(args):
			i++
			cfg.Host = args[i]
		// 解析 --host=HOST 形式
		case strings.HasPrefix(arg, "--host="):
			cfg.Host = arg[len("--host="):]
		}
	}

	// 初始化文件日志器
	logDir := trace.DefaultTraceDir()
	if cfg.DBPath != "" && cfg.DBPath != "backend.db" {
		logDir = filepath.Dir(cfg.DBPath)
	}
	if err := logger.Init(logDir, false, logger.DEBUG); err != nil {
		log.Printf("warning: logger init failed: %v", err)
	}
	defer logger.Close()
	logger.Info("backend.cmd", "backend starting: host=%s port=%d db=%s", cfg.Host, cfg.Port, cfg.DBPath)

	// 根据配置创建后端服务器实例
	srv, err := backend.NewServer(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}
	// 函数退出时确保关闭服务器资源
	defer srv.Close()

	// 启动服务器并开始监听请求
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

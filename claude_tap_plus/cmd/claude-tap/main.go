package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/config"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/session"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// main 是 claude-tap-plus 的入口函数，负责解析子命令并分发到对应处理函数。
//
// 支持的子命令：
//   session-push    收集本地 Claude 会话到存储
//   session-pull    从存储恢复会话到 ~/.claude/
//   session-status  显示会话存储状态
//   backend         启动后端服务器模式
//   help            显示使用帮助
//
// 无子命令时默认进入代理模式（拦截 Claude Code API 流量）。
func main() {
	// 设置日志格式：仅显示时间（含微秒）
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	// 初始化文件日志器（使用默认 trace 目录）
	logDir := trace.DefaultTraceDir()
	if err := logger.Init(logDir, false, logger.INFO); err != nil {
		log.Printf("warning: logger init failed: %v", err)
	}
	defer logger.Close()

	// 检测子命令，如果第一个参数是已知子命令则分发执行
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "session-push":
			runSessionPush(os.Args[2:])
			return
		case "session-pull":
			runSessionPull(os.Args[2:])
			return
		case "session-status":
			runSessionStatus(os.Args[2:])
			return
		case "backend":
			runBackend(os.Args[2:])
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// 默认进入代理模式，拦截并记录 Claude Code 的 API 请求
	runProxy(os.Args[1:])
}

// printUsage 打印命令行使用帮助信息到标准输出。
func printUsage() {
	fmt.Println("claude-tap-plus — Claude Code proxy, trace recorder, and session sync")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  claude-tap-plus [flags] [-- claude-args...]      Run proxy mode (default)")
	fmt.Println("  claude-tap-plus session-push [flags]             Collect sessions to local storage")
	fmt.Println("  claude-tap-plus session-pull [flags]             Restore sessions to ~/.claude/")
	fmt.Println("  claude-tap-plus session-status [flags]           Show session storage state")
	fmt.Println()
	fmt.Println("Proxy flags:")
	fmt.Println("  --tap-target URL       Upstream API target (default: auto-detect)")
	fmt.Println("  --tap-port PORT        Local proxy port (default: random)")
	fmt.Println("  --tap-output-dir DIR   Trace output directory")
	fmt.Println("  --tap-verbose          Enable verbose logging")
	fmt.Println("  --tap-console          Output logs to console (requires --tap-verbose)")
	fmt.Println("  --tap-profile NAME     Use named profile from profiles.json")
	fmt.Println("  --tap-api-key KEY      Override API key (highest priority)")
	fmt.Println("  --tap-auth-token TOKEN Override OAuth token (highest priority)")
	fmt.Println("  --tap-base-url URL     Override upstream API URL (highest priority)")
	fmt.Println("  --claude               Alias for -- (pass remaining args to claude)")
	fmt.Println()
	fmt.Println("Session-push flags:")
	fmt.Println("  --all                  Collect all projects")
	fmt.Println("  --force                Overwrite existing files")
	fmt.Println("  --dry-run              Preview only")
	fmt.Println("  --project NAME         Override project name")
	fmt.Println()
	fmt.Println("Session-pull flags:")
	fmt.Println("  --all                  Restore all projects")
	fmt.Println("  --project NAME         Override project name")
	fmt.Println("  --dry-run              Preview only")
	fmt.Println()
	fmt.Println("Session-status flags:")
	fmt.Println("  --verbose              Show file-level details")
}

// runProxy 是默认的代理模式主逻辑。
//
// 工作流程：
//  1. 解析 --tap-* 系列标志，提取剩余的参数传递给 Claude Code
//  2. 自动探测上游 API 目标地址
//  3. 启动本地反向代理监听指定端口
//  4. 解析 Claude Code 可执行文件路径
//  5. 构建子进程环境变量（注入代理地址）
//  6. 启动 Claude Code 作为子进程
//  7. 监听系统信号并转发给子进程，支持优雅关闭
//  8. 子进程退出后打印 API 调用统计摘要
func runProxy(args []string) {
	// 记录原始命令行参数，用于生成 resume 命令
	originalArgs := append([]string{}, args...)

	// 解析 --tap-* 标志，收集剩余参数作为传递给 Claude Code 的参数
	var (
		tapTarget    string // 上游 API 目标地址
		tapPort      int    // 本地代理监听端口
		tapOutputDir string // 追踪记录输出目录
		tapVerbose   bool   // 是否启用详细日志
		tapConsole   bool   // 是否输出日志到终端（需配合 --tap-verbose）
		tapProfile   string // 配置名（读取 profiles.json）
		tapAPIKey    string // 直接传入 API Key
		tapAuthToken string // 直接传入 OAuth Token
		tapBaseURL   string // 直接传入上游地址
	)

	var claudeArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		// 解析 --tap-target 参数（空格分隔形式）
		case arg == "--tap-target" && i+1 < len(args):
			i++
			tapTarget = args[i]
		// 解析 --tap-target=URL 形式
		case strings.HasPrefix(arg, "--tap-target="):
			tapTarget = arg[len("--tap-target="):]
		// 解析 --tap-port 参数（空格分隔形式）
		case arg == "--tap-port" && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &tapPort)
		// 解析 --tap-port=PORT 形式
		case strings.HasPrefix(arg, "--tap-port="):
			fmt.Sscanf(arg[len("--tap-port="):], "%d", &tapPort)
		// 解析 --tap-output-dir 参数（空格分隔形式）
		case arg == "--tap-output-dir" && i+1 < len(args):
			i++
			tapOutputDir = args[i]
		// 解析 --tap-output-dir=DIR 形式
		case strings.HasPrefix(arg, "--tap-output-dir="):
			tapOutputDir = arg[len("--tap-output-dir="):]
		// 启用详细日志
		case arg == "--tap-verbose":
			tapVerbose = true
		// 启用终端日志输出（需配合 --tap-verbose）
		case arg == "--tap-console":
			tapConsole = true
		// 解析 --tap-profile 参数（读取 profiles.json 中的配置）
		case arg == "--tap-profile" && i+1 < len(args):
			i++
			tapProfile = args[i]
		case strings.HasPrefix(arg, "--tap-profile="):
			tapProfile = arg[len("--tap-profile="):]
		// 解析 --tap-api-key 参数（直接传入 API Key）
		case arg == "--tap-api-key" && i+1 < len(args):
			i++
			tapAPIKey = args[i]
		case strings.HasPrefix(arg, "--tap-api-key="):
			tapAPIKey = arg[len("--tap-api-key="):]
		// 解析 --tap-auth-token 参数（直接传入 OAuth Token）
		case arg == "--tap-auth-token" && i+1 < len(args):
			i++
			tapAuthToken = args[i]
		case strings.HasPrefix(arg, "--tap-auth-token="):
			tapAuthToken = arg[len("--tap-auth-token="):]
		// 解析 --tap-base-url 参数（直接传入上游地址，优先级高于 --tap-target）
		case arg == "--tap-base-url" && i+1 < len(args):
			i++
			tapBaseURL = args[i]
		case strings.HasPrefix(arg, "--tap-base-url="):
			tapBaseURL = arg[len("--tap-base-url="):]
		// -- 或 --claude 或 claude 后的所有参数都传递给 Claude Code
		case arg == "--", arg == "--claude", arg == "claude":
			claudeArgs = append(claudeArgs, args[i+1:]...)
			i = len(args) // 停止解析
		// 未知的 --tap-* 标志，直接忽略
		case strings.HasPrefix(arg, "--tap-"):
		// 其他参数视为 Claude Code 的参数
		default:
			claudeArgs = append(claudeArgs, arg)
		}
	}

	// 如果未指定输出目录，使用默认追踪目录
	if tapOutputDir == "" {
		tapOutputDir = trace.DefaultTraceDir()
	}

	// 校验：--tap-console 必须配合 --tap-verbose
	if tapConsole && !tapVerbose {
		fmt.Fprintf(os.Stderr, "Error: --tap-console requires --tap-verbose\n")
		os.Exit(1)
	}

	// 根据详细程度重新初始化日志器
	if tapVerbose {
		logger.Init(tapOutputDir, tapConsole, logger.DEBUG)
	} else {
		logger.Init(tapOutputDir, false, logger.INFO)
	}
	logger.Info("main", "proxy mode: output_dir=%s verbose=%v", tapOutputDir, tapVerbose)

	// 按优先级解析上游 API 配置（--tap-base-url > --tap-target > profile > env > ~/.claude.json > default）
	resolved, err := config.ResolveTargetConfig(
		firstNonEmpty(tapBaseURL, tapTarget),
		tapAPIKey,
		tapAuthToken,
		tapProfile,
		&config.ClaudeClient,
	)
	if err != nil {
		log.Fatalf("resolve config: %v", err)
	}
	target := resolved.BaseURL
	if tapVerbose {
		log.Printf("upstream target: %s", target)
	}

	// 创建反向代理实例，绑定上游目标和追踪输出目录
	rp := proxy.NewReverseProxy(target, tapOutputDir)
	port := tapPort
	// 启动代理监听（如果 port 为 0 则随机分配端口）
	actualPort, err := rp.Start("127.0.0.1", port)
	if err != nil {
		log.Fatalf("start proxy: %v", err)
	}
	// 函数退出时停止代理
	defer rp.Stop()

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", actualPort)
	log.Printf("proxy listening on %s", proxyURL)

	// 自动检测并启动后端服务（如果未运行）
	ensureBackend()

	// 向后端注册本代理（hooks 通过后端转发 trace-init）
	registerProxyWithBackend(proxyURL)
	// 代理退出时注销
	defer unregisterProxyFromBackend()

	// 设置会话初始化回调：注册到 proxy.json
	rp.OnSessionInit = func(sessionID, projectSlug string) {
		key := ProxySessionKey(projectSlug, sessionID)
		if err := RegisterProxySession(key, time.Now().Format(time.RFC3339)); err != nil {
			logger.Warn("main", "register proxy session failed: %v", err)
		}
	}
	// 代理退出时注销 proxy.json 中的会话
	defer func() {
		if sid := rp.SessionID(); sid != "" {
			slug := rp.ProjectSlug()
			key := ProxySessionKey(slug, sid)
			if err := UnregisterProxySession(key); err != nil {
				logger.Warn("main", "unregister proxy session failed: %v", err)
			}
		}
	}()

	// 解析 Claude Code 可执行文件路径
	claudePath, err := config.ResolveCmd(&config.ClaudeClient)
	if err != nil {
		log.Fatal(err)
	}

	// 构建子进程环境变量，将 API 请求指向本地代理
	childEnv := config.BuildChildEnv(&config.ClaudeClient, proxyURL)

	// 注入认证信息：有 API Key 用 API Key，有 Token 用 Token，互斥避免冲突
	if resolved.APIKey != "" {
		childEnv = removeEnvVar(childEnv, "ANTHROPIC_AUTH_TOKEN")
		childEnv = append(childEnv, "ANTHROPIC_API_KEY="+resolved.APIKey)
	} else if resolved.AuthToken != "" {
		childEnv = removeEnvVar(childEnv, "ANTHROPIC_API_KEY")
		childEnv = append(childEnv, "ANTHROPIC_AUTH_TOKEN="+resolved.AuthToken)
	}

	// 对于需要注入 --settings 参数的客户端（如 Claude Code），自动添加
	finalArgs := claudeArgs
	if config.ClaudeClient.InjectSettingsEnv && !config.HasSettingsArg(claudeArgs) {
		settingsArgs := config.BuildSettingsArgs(&config.ClaudeClient, proxyURL)
		finalArgs = append(settingsArgs, claudeArgs...)
	}

	// 启动 Claude Code 作为子进程
	cmd := exec.Command(claudePath, finalArgs...)
	cmd.Env = childEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("\n🚀 Starting Claude Code: claude %s\n", formatArgs(claudeArgs))
	fmt.Printf("   ANTHROPIC_BASE_URL=%s\n\n", proxyURL)

	if err := cmd.Start(); err != nil {
		log.Fatalf("start claude: %v", err)
	}

	// 监听系统信号（SIGINT、SIGTERM），转发给子进程以实现优雅关闭
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sigCount := 0
		for sig := range sigChan {
			sigCount++
			if sigCount == 1 {
				// 第一次收到信号：发送给子进程请求优雅退出
				fmt.Printf("\n⏳ Shutting down Claude Code... (Ctrl+C again to force)\n")
				_ = cmd.Process.Signal(sig)
			} else {
				// 再次收到信号：强制终止子进程
				_ = cmd.Process.Kill()
			}
		}
	}()

	// 等待子进程结束
	waitErr := cmd.Wait()

	// 停止信号处理器，释放资源
	signal.Stop(sigChan)
	close(sigChan)

	// 打印运行摘要：API 调用次数和 token 使用情况
	summary := rp.Summary()
	tracePath := rp.TracePath()

	fmt.Printf("\n📋 Claude Code exited")
	if waitErr != nil {
		fmt.Printf(" with error: %v", waitErr)
	}
	fmt.Printf("\n")
	fmt.Printf("   API calls:      %v\n", summary["api_calls"])
	fmt.Printf("   Input tokens:    %v\n", summary["input_tokens"])
	fmt.Printf("   Output tokens:   %v\n", summary["output_tokens"])
	fmt.Printf("   Cache read:      %v\n", summary["cache_read_tokens"])
	fmt.Printf("   Cache create:    %v\n", summary["cache_create_tokens"])
	if tracePath != "" {
		fmt.Printf("   Trace: %s\n", shortenPath(tracePath))
	}
	fmt.Printf("\n")

	// 生成 Resume 命令（仅在有 session_id 时）
	sessionID := rp.SessionID()
	if sessionID != "" {
		resumeCmd := buildResumeCommand(originalArgs, sessionID)
		fmt.Printf("📎 Resume:\n")
		fmt.Printf("   %s\n", resumeCmd)
	}
}

// runSessionPush 执行 session-push 子命令，收集本地 Claude 会话到集中存储。
//
// 支持的参数：
//   --all       收集所有项目的会话
//   --force     覆盖已存在的会话文件
//   --dry-run   仅预览，不实际执行
//   --project   指定项目名称，覆盖自动检测
func runSessionPush(args []string) {
	var opts session.PushOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--all":
			opts.All = true
		case arg == "--force":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--project" && i+1 < len(args):
			i++
			opts.Project = args[i]
		case strings.HasPrefix(arg, "--project="):
			opts.Project = arg[len("--project="):]
		}
	}

	fmt.Println("📥 Collecting Claude sessions...")
	results, err := session.SessionPush(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 汇总各项目的收集结果
	fmt.Println()
	totalNew, totalSkipped := 0, 0
	for _, r := range results {
		totalNew += r.SessionsNew
		totalSkipped += r.SessionsSkipped
		if r.SessionsNew > 0 || r.SessionsSkipped > 0 {
			fmt.Printf("  %s: %d new, %d skipped\n", r.Project, r.SessionsNew, r.SessionsSkipped)
		}
	}
	fmt.Printf("Total: %d sessions collected, %d skipped\n", totalNew, totalSkipped)
}

// runSessionPull 执行 session-pull 子命令，从集中存储恢复会话到本地 ~/.claude/ 目录。
//
// 支持的参数：
//   --all       恢复所有项目的会话
//   --dry-run   仅预览，不实际执行
//   --project   指定项目名称，覆盖自动检测
func runSessionPull(args []string) {
	var opts session.PullOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--all":
			opts.All = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--project" && i+1 < len(args):
			i++
			opts.Project = args[i]
		case strings.HasPrefix(arg, "--project="):
			opts.Project = arg[len("--project="):]
		}
	}

	fmt.Println("📤 Restoring sessions...")
	results, err := session.SessionPull(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 汇总各项目的恢复结果
	fmt.Println()
	totalRestored, totalSkipped := 0, 0
	for _, r := range results {
		totalRestored += r.SessionsRestored
		totalSkipped += r.SessionsSkipped
		if r.SessionsRestored > 0 || r.SessionsSkipped > 0 {
			fmt.Printf("  %s: %d restored, %d skipped\n", r.Project, r.SessionsRestored, r.SessionsSkipped)
		}
	}
	fmt.Printf("Total: %d sessions restored, %d skipped\n", totalRestored, totalSkipped)
}

// runSessionStatus 执行 session-status 子命令，显示会话存储的当前状态。
//
// 支持的参数：
//   --verbose   显示文件级别的详细信息
func runSessionStatus(args []string) {
	var opts session.StatusOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--verbose":
			opts.Verbose = true
		}
	}

	if err := session.SessionStatus(opts); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// formatArgs 将字符串切片格式化为带空格分隔的命令行参数字符串。
// 如果参数中包含空格或制表符，则使用引号包裹。
func formatArgs(args []string) string {
	result := ""
	for _, a := range args {
		if containsSpace(a) {
			result += fmt.Sprintf(" %q", a)
		} else {
			result += " " + a
		}
	}
	return strings.TrimSpace(result)
}

// containsSpace 检查字符串 s 中是否包含空格或制表符。
func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' {
			return true
		}
	}
	return false
}

// firstNonEmpty 返回第一个非空字符串。如果所有参数都为空，返回空字符串。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// buildResumeCommand 根据原始命令行参数和新的 session_id 生成 resume 命令。
//
// 逻辑：
//  1. 复制原始参数
//  2. 移除已有的 --resume <old-id>（如果存在）
//  3. 追加 --resume <new-session-id>
func buildResumeCommand(originalArgs []string, sessionID string) string {
	filtered := make([]string, 0, len(originalArgs))
	skip := false
	for i, arg := range originalArgs {
		if skip {
			skip = false
			continue
		}
		if arg == "--resume" && i+1 < len(originalArgs) {
			skip = true // 跳过 --resume 及其值
			continue
		}
		if strings.HasPrefix(arg, "--resume=") {
			continue // 跳过 --resume=id 形式
		}
		filtered = append(filtered, arg)
	}
	filtered = append(filtered, "--resume", sessionID)
	return "claude-tap-plus " + formatArgs(filtered)
}

// shortenPath 将路径中的用户主目录替换为 ~，使输出更简洁。
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// removeEnvVar 从环境变量列表中移除指定名称的变量。
func removeEnvVar(env []string, name string) []string {
	prefix := name + "="
	result := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			result = append(result, e)
		}
	}
	return result
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/config"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/session"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	// Detect subcommand.
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
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Default: proxy mode.
	runProxy(os.Args[1:])
}

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

// runProxy is the default proxy mode — intercepts Claude Code API traffic.
func runProxy(args []string) {
	// Parse --tap-* flags, collect remaining args for claude.
	var (
		tapTarget    string
		tapPort      int
		tapOutputDir string
		tapVerbose   bool
	)

	var claudeArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--tap-target" && i+1 < len(args):
			i++
			tapTarget = args[i]
		case strings.HasPrefix(arg, "--tap-target="):
			tapTarget = arg[len("--tap-target="):]
		case arg == "--tap-port" && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &tapPort)
		case strings.HasPrefix(arg, "--tap-port="):
			fmt.Sscanf(arg[len("--tap-port="):], "%d", &tapPort)
		case arg == "--tap-output-dir" && i+1 < len(args):
			i++
			tapOutputDir = args[i]
		case strings.HasPrefix(arg, "--tap-output-dir="):
			tapOutputDir = arg[len("--tap-output-dir="):]
		case arg == "--tap-verbose":
			tapVerbose = true
		case arg == "--", arg == "--claude":
			// Everything after -- (or --claude) goes to claude.
			claudeArgs = append(claudeArgs, args[i+1:]...)
			i = len(args) // stop parsing
		case strings.HasPrefix(arg, "--tap-"):
			// Unknown tap flag, skip.
		default:
			claudeArgs = append(claudeArgs, arg)
		}
	}

	if tapOutputDir == "" {
		tapOutputDir = trace.DefaultTraceDir()
	}

	// Detect upstream target.
	target := tapTarget
	if target == "" {
		target = config.DetectTarget(&config.ClaudeClient)
	}
	if tapVerbose {
		log.Printf("upstream target: %s", target)
	}

	// Setup trace writer.
	tracePath := trace.NewTracePath(tapOutputDir)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		log.Fatalf("create trace writer: %v", err)
	}
	defer writer.Close()

	if tapVerbose {
		log.Printf("trace file: %s", tracePath)
	}

	// Start reverse proxy.
	rp := proxy.NewReverseProxy(target, writer)
	port := tapPort
	actualPort, err := rp.Start("127.0.0.1", port)
	if err != nil {
		log.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", actualPort)
	log.Printf("proxy listening on %s", proxyURL)

	// Resolve claude binary.
	claudePath, err := config.ResolveCmd(&config.ClaudeClient)
	if err != nil {
		log.Fatal(err)
	}

	// Build child environment.
	childEnv := config.BuildChildEnv(&config.ClaudeClient, proxyURL)

	// Inject --settings for clients that need it (e.g. Claude Code).
	finalArgs := claudeArgs
	if config.ClaudeClient.InjectSettingsEnv && !config.HasSettingsArg(claudeArgs) {
		settingsArgs := config.BuildSettingsArgs(&config.ClaudeClient, proxyURL)
		finalArgs = append(settingsArgs, claudeArgs...)
	}

	// Launch claude as child process.
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

	// Handle signals: forward to child process.
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sigCount := 0
		for sig := range sigChan {
			sigCount++
			if sigCount == 1 {
				fmt.Printf("\n⏳ Shutting down Claude Code... (Ctrl+C again to force)\n")
				_ = cmd.Process.Signal(sig)
			} else {
				_ = cmd.Process.Kill()
			}
		}
	}()

	// Wait for child to exit.
	waitErr := cmd.Wait()

	// Stop signal handler.
	signal.Stop(sigChan)
	close(sigChan)

	// Print summary.
	summary := writer.Summary()
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
	fmt.Printf("   Trace: %s\n", tracePath)
}

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

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' {
			return true
		}
	}
	return false
}

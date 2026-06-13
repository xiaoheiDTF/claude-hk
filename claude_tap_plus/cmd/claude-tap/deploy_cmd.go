package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/deploy"
)

// runDeploy 处理 deploy 子命令：将模板目录部署到目标项目。
func runDeploy(args []string) {
	var (
		target string
		dryRun bool
	)

	// 解析 flags（与 runBackend 相同的手动解析模式）
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case (arg == "--target" || arg == "-t") && i+1 < len(args):
			i++
			target = args[i]
		case strings.HasPrefix(arg, "--target="):
			target = arg[len("--target="):]
		case arg == "--dry-run":
			dryRun = true
		}
	}

	if target == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定 --target 参数\n")
		fmt.Fprintf(os.Stderr, "用法: claude-tap-plus deploy --target <项目路径>\n")
		os.Exit(1)
	}

	// 解析模板目录
	templateDir, err := deploy.TemplateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 执行部署
	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  target,
		TemplateDir: templateDir,
		DryRun:      dryRun,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	if dryRun {
		fmt.Println("=== 部署预览 (dry-run) ===")
		for _, p := range result.Plan {
			fmt.Printf("  [%s] %s — %s\n", p.Action, p.RelPath, p.Reason)
		}
		fmt.Printf("\n共 %d 个文件\n", len(result.Plan))
		return
	}

	fmt.Printf("部署完成: %d 个文件已复制, %d 个文件已跳过\n", result.FilesCopied, result.FilesSkipped)
	if result.BackupPath != "" {
		fmt.Printf("备份位置: %s\n", result.BackupPath)
	}
}

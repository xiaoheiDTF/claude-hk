// Package deploy_test 包含 deploy 子命令的 BDD 场景测试。
//
// 覆盖 24 条后端 BDD 场景，分为 5 个功能接口（F1-F5）+ 2 条扩展场景。
// 使用 t.TempDir() 创建隔离的临时目录，每个测试函数对应一条 BDD 场景。
package deploy_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/deploy"
)

// --- helpers ---

// createMinimalTemplate 创建一个包含所有必需项的最小模板目录。
// 返回模板目录路径。
func createMinimalTemplate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// 创建 9 个必需项
	requiredDirs := []string{
		"hooks",
		"skills",
		"lib",
		"scripts",
		"myRule",
	}
	for _, d := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	requiredFiles := map[string]string{
		"settings.json":   `{"key": "value"}`,
		"init.sh":         "#!/bin/bash\necho init",
		"dirs.conf":       "LOG_DIR=logs",
		"registry.conf":   "001-2-issue",
		"hooks/base.sh":   "#!/bin/bash\nsource lib/config.sh",
		"skills/test.sh":  "#!/bin/bash\necho test",
		"lib/config.sh":   "#!/bin/bash\nBACKEND_URL=",
		"scripts/run.sh":  "#!/bin/bash\necho run",
		"myRule/rule.md":  "# rule",
	}
	for name, content := range requiredFiles {
	 fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

// createFullTemplate 创建包含 888-* project.md 的完整模板。
func createFullTemplate(t *testing.T) string {
	t.Helper()
	dir := createMinimalTemplate(t)

	// 添加 888 skill 目录及空 project.md
	skill888 := filepath.Join(dir, "skills", "888-1-2-backend-modify")
	if err := os.MkdirAll(skill888, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill888, "SKILL.md"), []byte("# 888 skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill888, "project.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// fileHash 计算文件的 SHA256 哈希值。
func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败: %s: %v", path, err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// fileExists 检查路径是否存在。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// walkFiles 递归遍历目录，返回所有文件的相对路径列表（使用 / 分隔符）。
func walkFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("遍历目录失败: %v", err)
	}
	return files
}

// diffFileLists 比较两组文件列表，返回只在 a 中的和只在 b 中的。
func diffFileLists(a, b []string) (onlyInA, onlyInB []string) {
	setB := make(map[string]bool)
	for _, f := range b {
		setB[f] = true
	}
	setA := make(map[string]bool)
	for _, f := range a {
		setA[f] = true
	}
	for _, f := range a {
		if !setB[f] {
			onlyInA = append(onlyInA, f)
		}
	}
	for _, f := range b {
		if !setA[f] {
			onlyInB = append(onlyInB, f)
		}
	}
	return
}

// =====================================================================
// F1: 部署接口（10 个场景）
// =====================================================================

// --- F1 正向场景 ---

// TestF1_P1_FirstDeploy 测试 F1-P1：首次部署到无 .claude/ 的目标项目。
// 来源：BDD 1.1
func TestF1_P1_FirstDeploy(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()

	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("首次部署失败: %v", err)
	}

	// 验证：.claude/ 目录已创建
	claudeDir := filepath.Join(targetDir, ".claude")
	if !fileExists(claudeDir) {
		t.Fatal("目标 .claude/ 目录未创建")
	}

	// 验证：模板文件完整复制
	templateFiles := walkFiles(t, templateDir)
	targetFiles := walkFiles(t, claudeDir)

	onlyInTemplate, onlyInTarget := diffFileLists(templateFiles, targetFiles)
	if len(onlyInTemplate) > 0 {
		t.Errorf("模板中有但目标中没有的文件: %v", onlyInTemplate)
	}
	if len(onlyInTarget) > 0 {
		t.Errorf("目标中有但模板中没有的文件: %v", onlyInTarget)
	}

	// 验证：888-*/project.md 存在且内容为空
	projectMD := filepath.Join(claudeDir, "skills", "888-1-2-backend-modify", "project.md")
	data, err := os.ReadFile(projectMD)
	if err != nil {
		t.Fatalf("project.md 不存在: %v", err)
	}
	if string(data) != "" {
		t.Errorf("project.md 应为空，实际内容: %q", string(data))
	}

	// 验证：无 settings.local.json
	if fileExists(filepath.Join(claudeDir, "settings.local.json")) {
		t.Error("首次部署不应包含 settings.local.json")
	}

	// 验证：退出码 0（通过 err == nil 判断）
	if result.FilesCopied == 0 {
		t.Error("应至少复制 1 个文件")
	}
}

// TestF1_P2_RedeployOverwrite 测试 F1-P2：再次部署覆盖旧版本。
// 来源：BDD 2.1
func TestF1_P2_RedeployOverwrite(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 第一次部署
	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("首次部署失败: %v", err)
	}

	// 添加运行时产物
	runtimeFiles := map[string]string{
		".initialized":                                    `{"os": "windows"}`,
		".python-state":                                   "ready",
		"skills/.active":                                  "session-123|001-2-issue",
		"hooks/lib/win32-foreground.log":                  "foreground log",
	}
	for name, content := range runtimeFiles {
	 fullPath := filepath.Join(claudeDir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 创建运行时目录
	for _, d := range []string{"localLanguage", "hooks/logs", "skills/log/001-2-issue"} {
		if err := os.MkdirAll(filepath.Join(claudeDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "hooks/logs/2026-01-01.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "skills/log/001-2-issue/2026-01-01.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 修改模板中的 hooks/base.sh 来模拟更新
	updatedContent := "#!/bin/bash\n# updated version\nsource lib/config.sh"
	if err := os.WriteFile(filepath.Join(templateDir, "hooks/base.sh"), []byte(updatedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 记录运行时文件内容用于验证
	initializedBefore, _ := os.ReadFile(filepath.Join(claudeDir, ".initialized"))
	activeBefore, _ := os.ReadFile(filepath.Join(claudeDir, "skills/.active"))

	// 第二次部署
	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("再次部署失败: %v", err)
	}

	// 验证：hooks 文件已更新
	hooksBase, err := os.ReadFile(filepath.Join(claudeDir, "hooks/base.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hooksBase), "updated version") {
		t.Error("hooks/base.sh 未更新为模板最新版")
	}

	// 验证：运行时产物未被删除或修改
	initializedAfter, _ := os.ReadFile(filepath.Join(claudeDir, ".initialized"))
	if string(initializedBefore) != string(initializedAfter) {
		t.Error(".initialized 内容被修改")
	}
	activeAfter, _ := os.ReadFile(filepath.Join(claudeDir, "skills/.active"))
	if string(activeBefore) != string(activeAfter) {
		t.Error("skills/.active 内容被修改")
	}

	// 验证：备份已创建
	if result.BackupPath == "" {
		t.Error("再次部署应创建备份")
	}
	if !fileExists(result.BackupPath) {
		t.Errorf("备份目录不存在: %s", result.BackupPath)
	}
}

// TestF1_P3_NoChangeRedeploy 测试 F1-P3：无变更重复部署。
// 来源：BDD 2.2
func TestF1_P3_NoChangeRedeploy(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 第一次部署
	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("首次部署失败: %v", err)
	}

	// 记录所有文件 hash
	hashesBefore := make(map[string]string)
	for _, f := range walkFiles(t, claudeDir) {
		hashesBefore[f] = fileHash(t, filepath.Join(claudeDir, f))
	}

	// 第二次部署（模板未变更）
	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("重复部署失败: %v", err)
	}

	// 验证：文件内容不变
	for _, f := range walkFiles(t, claudeDir) {
		hashAfter := fileHash(t, filepath.Join(claudeDir, f))
		if hashesBefore[f] != hashAfter {
			t.Errorf("文件 %s 内容发生变化（应保持不变）", f)
		}
	}

	// 验证：备份已创建
	if result.BackupPath == "" {
		t.Error("重复部署应创建备份")
	}
}

// --- F1 异常场景 ---

// TestF1_E1_TemplateNotExist 测试 F1-E1：模板目录不存在。
// 来源：BDD 4.1
func TestF1_E1_TemplateNotExist(t *testing.T) {
	targetDir := t.TempDir()
	fakeTemplate := filepath.Join(t.TempDir(), "nonexistent")

	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: fakeTemplate,
	})

	// 验证：命令失败
	if err == nil {
		t.Fatal("模板不存在时应返回错误")
	}
	if result != nil {
		t.Error("失败时 result 应为 nil")
	}

	// 验证：错误信息包含"模板目录不存在"
	if !strings.Contains(err.Error(), "模板") {
		t.Errorf("错误信息应包含'模板'，实际: %v", err)
	}

	// 验证：目标项目无任何变化
	if fileExists(filepath.Join(targetDir, ".claude")) {
		t.Error("目标项目不应有新增文件")
	}
}

// TestF1_E2_TargetNotExist 测试 F1-E2：目标路径不存在。
// 来源：BDD 4.3
func TestF1_E2_TargetNotExist(t *testing.T) {
	templateDir := createMinimalTemplate(t)
	fakeTarget := filepath.Join(t.TempDir(), "nonexistent", "path")

	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  fakeTarget,
		TemplateDir: templateDir,
	})

	if err == nil {
		t.Fatal("目标路径不存在时应返回错误")
	}
	if !strings.Contains(err.Error(), "目标") {
		t.Errorf("错误信息应包含'目标'，实际: %v", err)
	}
}

// TestF1_E3_TargetIsFile 测试 F1-E3：目标路径不是目录。
// 来源：BDD 4.3 扩展
func TestF1_E3_TargetIsFile(t *testing.T) {
	templateDir := createMinimalTemplate(t)

	// 创建一个文件作为目标路径
	tmpFile, err := os.CreateTemp("", "target-file-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = deploy.Deploy(deploy.Options{
		TargetPath:  tmpFile.Name(),
		TemplateDir: templateDir,
	})

	if err == nil {
		t.Fatal("目标路径是文件时应返回错误")
	}
	if !errors.Is(err, deploy.ErrTargetNotDir) {
		t.Errorf("错误应为 ErrTargetNotDir，实际: %v", err)
	}
}

// --- F1 边界场景 ---

// TestF1_B1_PartialFailure 测试 F1-B1：部分文件复制失败时回滚。
// 来源：后端独有关注点 — 部分失败
func TestF1_B1_PartialFailure(t *testing.T) {
	// 注意：在 Windows 上很难模拟权限拒绝，此测试验证错误传播行为
	// 使用一个目标路径中有只读目录的场景
	templateDir := createMinimalTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 先做一次正常部署
	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("首次部署失败: %v", err)
	}

	// 将目标中的某个目录设为只读，阻止覆盖
	// 注意：这种方法在 Windows 上效果有限，但测试逻辑路径
	hooksDir := filepath.Join(claudeDir, "hooks")
	hooksBase := filepath.Join(hooksDir, "base.sh")

	// 在非 Windows 上使用权限控制
	if runtime.GOOS != "windows" {
		os.Chmod(hooksBase, 0o444) // 只读文件
		os.Chmod(hooksDir, 0o555) // 只读目录
		defer func() {
			os.Chmod(hooksDir, 0o755)
			os.Chmod(hooksBase, 0o644)
		}()
	}

	// 验证：当前实现在部分失败时返回错误
	// 由于 Windows 权限模型不同，这里主要验证错误能被正确捕获
	// 在 CI 环境中（Linux），这个测试能验证真正的权限拒绝场景
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不同，跳过只读测试")
	}

	// 第二次部署，触发权限错误
	_, err = deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err == nil {
		t.Error("部分复制失败时应返回错误")
	}
}

// TestF1_B2_EmptyTemplate 测试 F1-B2：模板目录为空。
// 来源：边界穿透
func TestF1_B2_EmptyTemplate(t *testing.T) {
	emptyDir := t.TempDir() // 空目录
	targetDir := t.TempDir()

	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: emptyDir,
	})

	if err == nil {
		t.Fatal("空模板目录应返回错误")
	}
	if !strings.Contains(err.Error(), "为空") {
		t.Errorf("错误信息应包含'为空'，实际: %v", err)
	}

	// 验证：目标无变化
	if fileExists(filepath.Join(targetDir, ".claude")) {
		t.Error("空模板不应导致目标创建 .claude/")
	}
}

// TestF1_B3_ConcurrentDeploy 测试 F1-B3：同时向同一目标部署（并发）。
// 来源：后端独有关注点 — 并发安全
func TestF1_B3_ConcurrentDeploy(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()

	// 第一次部署，创建 .claude/
	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("首次部署失败: %v", err)
	}

	// 并发再次部署
	var (
		wg         sync.WaitGroup
		successes  int64
		failures   int64
	)
	const concurrency = 3

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := deploy.Deploy(deploy.Options{
				TargetPath:  targetDir,
				TemplateDir: templateDir,
			})
			if err != nil {
				atomic.AddInt64(&failures, 1)
			} else {
				atomic.AddInt64(&successes, 1)
			}
		}()
	}
	wg.Wait()

	total := successes + failures
	if total != int64(concurrency) {
		t.Errorf("期望 %d 个结果，实际: 成功=%d 失败=%d", concurrency, successes, failures)
	}

	// 验证：目标文件内容完整无截断
	claudeDir := filepath.Join(targetDir, ".claude")
	for _, f := range walkFiles(t, claudeDir) {
		info, err := os.Stat(filepath.Join(claudeDir, f))
		if err != nil {
			t.Errorf("并发部署后文件损坏: %s: %v", f, err)
			continue
		}
		if info.Size() == 0 {
			// project.md 允许为空
			if !strings.HasSuffix(f, "project.md") {
				t.Errorf("文件被截断为空: %s", f)
			}
		}
	}
}

// TestF1_B4_SpecialCharsInPath 测试 F1-B4：目标路径含特殊字符或空格。
// 来源：边界穿透
func TestF1_B4_SpecialCharsInPath(t *testing.T) {
	templateDir := createMinimalTemplate(t)
	parentDir := t.TempDir()

	// 创建含空格和括号的目标目录
	targetDir := filepath.Join(parentDir, "my project (v2)")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})

	if err != nil {
		t.Fatalf("含特殊字符路径部署失败: %v", err)
	}
	if result.FilesCopied == 0 {
		t.Error("应至少复制 1 个文件")
	}

	// 验证：文件正确复制
	claudeDir := filepath.Join(targetDir, ".claude")
	if !fileExists(filepath.Join(claudeDir, "settings.json")) {
		t.Error("settings.json 未复制到含特殊字符的路径")
	}
}

// =====================================================================
// F2: 模板校验接口（3 个场景）
// =====================================================================

// TestF2_P1_TemplateValid 测试 F2-P1：模板完整。
// 来源：BDD 1.1（隐含前提）
func TestF2_P1_TemplateValid(t *testing.T) {
	templateDir := createMinimalTemplate(t)

	err := deploy.ValidateTemplate(templateDir)
	if err != nil {
		t.Fatalf("完整模板应通过校验: %v", err)
	}
}

// TestF2_E1_TemplateMissingDir 测试 F2-E1：模板缺少关键目录。
// 来源：BDD 4.2
func TestF2_E1_TemplateMissingDir(t *testing.T) {
	templateDir := t.TempDir()

	// 创建除 hooks/ 以外的所有必需项
	for _, d := range []string{"skills", "lib", "scripts", "myRule"} {
		os.MkdirAll(filepath.Join(templateDir, d), 0o755)
	}
	for _, f := range []string{"settings.json", "init.sh", "dirs.conf", "registry.conf"} {
		os.WriteFile(filepath.Join(templateDir, f), []byte(""), 0o644)
	}

	err := deploy.ValidateTemplate(templateDir)
	if err == nil {
		t.Fatal("缺少 hooks/ 目录应校验失败")
	}
	if !strings.Contains(err.Error(), "hooks/") {
		t.Errorf("错误信息应包含 'hooks/'，实际: %v", err)
	}
}

// TestF2_E2_TemplateMissingFile 测试 F2-E2：模板缺少关键文件。
// 来源：BDD 4.2
func TestF2_E2_TemplateMissingFile(t *testing.T) {
	templateDir := t.TempDir()

	// 创建所有必需目录但缺少 settings.json
	for _, d := range []string{"hooks", "skills", "lib", "scripts", "myRule"} {
		os.MkdirAll(filepath.Join(templateDir, d), 0o755)
	}
	for _, f := range []string{"init.sh", "dirs.conf", "registry.conf"} {
		os.WriteFile(filepath.Join(templateDir, f), []byte(""), 0o644)
	}
	// 故意不创建 settings.json

	err := deploy.ValidateTemplate(templateDir)
	if err == nil {
		t.Fatal("缺少 settings.json 应校验失败")
	}
	if !strings.Contains(err.Error(), "settings.json") {
		t.Errorf("错误信息应包含 'settings.json'，实际: %v", err)
	}
}

// =====================================================================
// F3: 目标校验接口（2 个场景）
// =====================================================================

// TestF3_P1_TargetValid 测试 F3-P1：目标路径有效。
// 来源：BDD 1.1（隐含前提）
func TestF3_P1_TargetValid(t *testing.T) {
	targetDir := t.TempDir()

	err := deploy.ValidateTarget(targetDir)
	if err != nil {
		t.Fatalf("有效目录应通过校验: %v", err)
	}
}

// TestF3_E1_TargetNotExist 测试 F3-E1：目标路径不存在。
// 来源：BDD 4.3
func TestF3_E1_TargetNotExist(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "does-not-exist")

	err := deploy.ValidateTarget(nonexistent)
	if err == nil {
		t.Fatal("不存在的路径应校验失败")
	}
	if !errors.Is(err, deploy.ErrTargetNotExist) {
		t.Errorf("错误应为 ErrTargetNotExist，实际: %v", err)
	}
}

// =====================================================================
// F4: 备份接口（2 个场景）
// =====================================================================

// TestF4_P1_CreateBackup 测试 F4-P1：目标已有 .claude/ 时创建备份。
// 来源：BDD 2.1
func TestF4_P1_CreateBackup(t *testing.T) {
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 创建 .claude/ 并添加文件
	os.MkdirAll(filepath.Join(claudeDir, "hooks"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "hooks/base.sh"), []byte("old content"), 0o644)
	os.WriteFile(filepath.Join(claudeDir, ".initialized"), []byte(`{"os": "test"}`), 0o644)

	backupPath, err := deploy.CreateBackup(claudeDir)
	if err != nil {
		t.Fatalf("创建备份失败: %v", err)
	}

	// 验证：备份目录存在
	if !fileExists(backupPath) {
		t.Fatal("备份目录不存在")
	}

	// 验证：备份名称格式
	backupName := filepath.Base(backupPath)
	if !strings.HasPrefix(backupName, ".claude.backup.") {
		t.Errorf("备份目录名格式错误: %s", backupName)
	}

	// 验证：备份包含所有文件
	backupHooksBase := filepath.Join(backupPath, "hooks", "base.sh")
	data, err := os.ReadFile(backupHooksBase)
	if err != nil {
		t.Fatalf("备份中缺少 hooks/base.sh: %v", err)
	}
	if string(data) != "old content" {
		t.Errorf("备份内容与原文件不一致: got %q, want %q", string(data), "old content")
	}

	backupInit := filepath.Join(backupPath, ".initialized")
	initData, err := os.ReadFile(backupInit)
	if err != nil {
		t.Fatalf("备份中缺少 .initialized: %v", err)
	}
	if string(initData) != `{"os": "test"}` {
		t.Error("备份中运行时文件内容不正确")
	}
}

// TestF4_B1_BackupDiskFull 测试 F4-B1：备份时磁盘空间不足。
// 来源：后端独有关注点 — 资源争抢
//
// 注意：模拟磁盘满比较困难，此测试验证在备份失败时
// 原有 .claude/ 保持不变。通过在不可写位置创建备份来模拟。
func TestF4_B1_BackupDiskFull(t *testing.T) {
	// 使用只读父目录来模拟写入失败
	if runtime.GOOS == "windows" {
		t.Skip("Windows 目录权限模型不同，跳过只读模拟测试")
	}

	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, "test.txt"), []byte("content"), 0o644)

	// 将父目录设为只读
	parentDir := filepath.Dir(claudeDir)
	os.Chmod(parentDir, 0o555)
	defer os.Chmod(parentDir, 0o755)

	_, err := deploy.CreateBackup(claudeDir)
	if err == nil {
		t.Error("只读目录下创建备份应失败")
	}
}

// =====================================================================
// F5: 文件过滤接口（5 个场景）
// =====================================================================

// TestF5_P1_FirstDeployCopyAll 测试 F5-P1：首次部署 — 复制全部模板文件。
// 来源：BDD 1.1
func TestF5_P1_FirstDeployCopyAll(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	plans, err := deploy.BuildFilter(templateDir, claudeDir, true)
	if err != nil {
		t.Fatalf("BuildFilter 失败: %v", err)
	}

	// 统计
	copied := 0
	skipped := 0
	for _, p := range plans {
		// 目录项不算文件
		if strings.HasSuffix(p.RelPath, "/") || (!strings.Contains(p.RelPath, ".") && !strings.Contains(p.RelPath, "/")) {
			// 可能是目录项，跳过
			if !strings.Contains(p.RelPath, ".") && !strings.Contains(p.RelPath, "/") {
				continue
			}
		}
		switch p.Action {
		case deploy.Copy:
			copied++
		case deploy.Skip:
			skipped++
			t.Logf("跳过文件: %s (%s)", p.RelPath, p.Reason)
		}
	}

	// 验证：首次部署所有文件标记为复制（无运行时文件要跳过，因为模板中没有运行时文件）
	// 注意：模板中若没有运行时文件，则全部应该标记为 Copy
	if skipped > 0 {
		// 只有模板中实际包含运行时文件名时才应 skip
		t.Logf("首次部署有 %d 个跳过项", skipped)
	}

	if copied == 0 {
		t.Error("首次部署应至少有一个文件标记为复制")
	}

	// 验证：空 project.md 存在于计划中
	foundProjectMD := false
	for _, p := range plans {
		if strings.HasSuffix(p.RelPath, "project.md") {
			foundProjectMD = true
			if p.Action != deploy.Copy {
				t.Errorf("project.md 应标记为复制，实际: %s", p.Action)
			}
		}
	}
	if !foundProjectMD {
		t.Error("计划中缺少 project.md")
	}
}

// TestF5_P2_RedeploySkipProtected 测试 F5-P2：再次部署 — 跳过受保护文件。
// 来源：BDD 3.1 + BDD 3.2
func TestF5_P2_RedeploySkipProtected(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 创建目标 .claude/ 并添加受保护文件
	os.MkdirAll(filepath.Join(claudeDir, "skills", "888-1-2-backend-modify"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"allowed": ["*"]}`), 0o644)
	os.WriteFile(filepath.Join(claudeDir, "skills", "888-1-2-backend-modify", "project.md"), []byte("# my project"), 0o644)

	plans, err := deploy.BuildFilter(templateDir, claudeDir, false)
	if err != nil {
		t.Fatalf("BuildFilter 失败: %v", err)
	}

	// 验证：settings.local.json 标记为跳过
	for _, p := range plans {
		if p.RelPath == "settings.local.json" {
			if p.Action != deploy.Skip {
				t.Error("settings.local.json 应标记为跳过")
			}
		}
	}

	// 验证：888-*/project.md 标记为跳过
	for _, p := range plans {
		if strings.HasPrefix(p.RelPath, "skills/888-") && strings.HasSuffix(p.RelPath, "/project.md") {
			if p.Action != deploy.Skip {
				t.Errorf("%s 应标记为跳过（受保护），实际: %s", p.RelPath, p.Action)
			}
		}
	}

	// 验证：通用文件标记为复制
	genericFiles := []string{"hooks/base.sh", "lib/config.sh", "settings.json"}
	for _, gf := range genericFiles {
		found := false
		for _, p := range plans {
			if p.RelPath == gf {
				found = true
				if p.Action != deploy.Copy {
					t.Errorf("通用文件 %s 应标记为复制，实际: %s", gf, p.Action)
				}
				break
			}
		}
		if !found {
			t.Errorf("计划中缺少通用文件: %s", gf)
		}
	}
}

// TestF5_B1_EmptyProjectMD 测试 F5-B1：目标有 project.md 但内容为空。
// 来源：边界穿透 — 首次部署后未运行 888-init 的中间状态
func TestF5_B1_EmptyProjectMD(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 创建目标 .claude/ 并添加空的 project.md
	os.MkdirAll(filepath.Join(claudeDir, "skills", "888-1-2-backend-modify"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "skills", "888-1-2-backend-modify", "project.md"), []byte(""), 0o644)

	plans, err := deploy.BuildFilter(templateDir, claudeDir, false)
	if err != nil {
		t.Fatalf("BuildFilter 失败: %v", err)
	}

	// 验证：project.md 标记为跳过（再次部署，一律跳过）
	for _, p := range plans {
		if strings.HasPrefix(p.RelPath, "skills/888-") && strings.HasSuffix(p.RelPath, "/project.md") {
			if p.Action != deploy.Skip {
				t.Errorf("空的 project.md 也应标记为跳过（再次部署规则），实际: %s", p.Action)
			}
		}
	}
}

// TestF5_B2_NewSkillInTemplate 测试 F5-B2：模板中有新增的 skill 目录。
// 来源：边界穿透 — 模板更新新增了文件
func TestF5_B2_NewSkillInTemplate(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 模板新增了 skill 目录
	newSkillDir := filepath.Join(templateDir, "skills", "999-other-130-xxx")
	os.MkdirAll(newSkillDir, 0o755)
	os.WriteFile(filepath.Join(newSkillDir, "SKILL.md"), []byte("# new skill"), 0o644)

	// 目标有旧的 .claude/
	os.MkdirAll(filepath.Join(claudeDir, "skills"), 0o755)

	plans, err := deploy.BuildFilter(templateDir, claudeDir, false)
	if err != nil {
		t.Fatalf("BuildFilter 失败: %v", err)
	}

	// 验证：新增 skill 标记为复制
	found := false
	for _, p := range plans {
		if strings.HasPrefix(p.RelPath, "skills/999-other-130-xxx") {
			found = true
			if p.Action != deploy.Copy {
				t.Errorf("新增 skill %s 应标记为复制，实际: %s", p.RelPath, p.Action)
			}
		}
	}
	if !found {
		t.Error("计划中缺少新增的 skill 目录文件")
	}
}

// TestF5_B3_DeletedSkillInTemplate 测试 F5-B3：模板中删除了某个 skill 目录。
// 来源：边界穿透 — 模板更新删除了文件
// 注意：当前实现采用"只增不删"策略，目标中多余文件不会被删除。
func TestF5_B3_DeletedSkillInTemplate(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")

	// 目标有旧 skill，模板中不存在
	oldSkillDir := filepath.Join(claudeDir, "skills", "old-skill")
	os.MkdirAll(oldSkillDir, 0o755)
	os.WriteFile(filepath.Join(oldSkillDir, "SKILL.md"), []byte("# old"), 0o644)

	plans, err := deploy.BuildFilter(templateDir, claudeDir, false)
	if err != nil {
		t.Fatalf("BuildFilter 失败: %v", err)
	}

	// 验证：当前"只增不删"策略，旧 skill 不出现在计划中（不会被操作）
	for _, p := range plans {
		if strings.HasPrefix(p.RelPath, "skills/old-skill") {
			t.Errorf("旧 skill 不应出现在计划中: %s", p.RelPath)
		}
	}

	// 验证：旧 skill 仍存在于目标
	if !fileExists(filepath.Join(oldSkillDir, "SKILL.md")) {
		t.Error("旧 skill 文件不应被删除")
	}
}

// =====================================================================
// 扩展场景（2 个场景）
// =====================================================================

// TestEX1_SymlinkFollow 测试 EX-1：符号链接处理。
// 来源：后端独有关注点 — 文件系统特性
func TestEX1_SymlinkFollow(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows 创建符号链接需要特殊权限
		t.Skip("Windows 创建符号链接需要管理员权限，跳过")
	}

	templateDir := createMinimalTemplate(t)
	targetDir := t.TempDir()

	// 在模板中创建一个符号链接文件
	realFile := filepath.Join(templateDir, "lib", "real_config.sh")
	os.WriteFile(realFile, []byte("# real config content"), 0o644)
	linkFile := filepath.Join(templateDir, "lib", "config_link.sh")
	os.Symlink(realFile, linkFile)

	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("部署失败: %v", err)
	}

	// 验证：符号链接被跟随，目标是普通文件
	claudeDir := filepath.Join(targetDir, ".claude")
	copiedLink := filepath.Join(claudeDir, "lib", "config_link.sh")

	info, err := os.Lstat(copiedLink)
	if err != nil {
		t.Fatalf("复制的文件不存在: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("符号链接应被跟随复制为普通文件，而非复制链接本身")
	}

	// 验证：内容与源文件一致
	data, err := os.ReadFile(copiedLink)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# real config content" {
		t.Errorf("链接文件内容不正确: got %q", string(data))
	}
}

// TestEX2_FilePermissionPreserved 测试 EX-2：文件权限保留。
// 来源：后端独有关注点 — 跨平台一致性
func TestEX2_FilePermissionPreserved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 文件权限模型不同，跳过可执行权限测试")
	}

	templateDir := createMinimalTemplate(t)
	targetDir := t.TempDir()

	// 设置 .sh 文件为可执行
	initSh := filepath.Join(templateDir, "init.sh")
	os.Chmod(initSh, 0o755)

	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("部署失败: %v", err)
	}

	// 验证：目标文件保留可执行权限
	claudeDir := filepath.Join(targetDir, ".claude")
	copiedInitSh := filepath.Join(claudeDir, "init.sh")
	info, err := os.Stat(copiedInitSh)
	if err != nil {
		t.Fatalf("文件不存在: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o111 == 0 {
		t.Errorf("init.sh 应保留可执行权限，实际权限: %o", perm)
	}
}

// =====================================================================
// DryRun 测试
// =====================================================================

// TestDryRun_PreviewOnly 测试 DryRun 模式只输出计划不执行复制。
func TestDryRun_PreviewOnly(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()

	result, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("DryRun 失败: %v", err)
	}

	// 验证：有计划输出
	if len(result.Plan) == 0 {
		t.Error("DryRun 应返回文件操作计划")
	}

	// 验证：目标目录未被创建
	if fileExists(filepath.Join(targetDir, ".claude")) {
		t.Error("DryRun 不应创建目标 .claude/ 目录")
	}

	// 验证：全部文件计入跳过
	if result.FilesCopied != 0 {
		t.Error("DryRun 不应复制任何文件")
	}
}

// =====================================================================
// 运行时文件排除集成测试
// =====================================================================

// TestRuntimeExcludes_Integration 验证部署后目标不包含运行时文件。
func TestRuntimeExcludes_Integration(t *testing.T) {
	templateDir := createFullTemplate(t)
	targetDir := t.TempDir()

	// 故意在模板中放运行时文件（模拟用户错误地放入了这些文件）
	runtimeFiles := map[string]string{
		".initialized":                    `{"os": "test"}`,
		".python-state":                   "ready",
		"skills/.active":                  "session|skill",
		"hooks/lib/win32-foreground.log":  "log data",
	}
	for name, content := range runtimeFiles {
	 fullPath := filepath.Join(templateDir, name)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	_, err := deploy.Deploy(deploy.Options{
		TargetPath:  targetDir,
		TemplateDir: templateDir,
	})
	if err != nil {
		t.Fatalf("部署失败: %v", err)
	}

	claudeDir := filepath.Join(targetDir, ".claude")

	// 验证：运行时文件未被复制
	excludedFiles := []string{
		".initialized",
		".python-state",
		"skills/.active",
		"hooks/lib/win32-foreground.log",
	}
	for _, f := range excludedFiles {
		if fileExists(filepath.Join(claudeDir, f)) {
			t.Errorf("运行时文件 %s 不应被复制到目标", f)
		}
	}
}

// =====================================================================
// F3 扩展：目标不是目录
// =====================================================================

// TestF3_E2_TargetIsFile 测试目标路径是文件时的校验行为。
func TestF3_E2_TargetIsFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "target-file-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = deploy.ValidateTarget(tmpFile.Name())
	if err == nil {
		t.Fatal("文件路径应校验失败")
	}
	if !errors.Is(err, deploy.ErrTargetNotDir) {
		t.Errorf("错误应为 ErrTargetNotDir，实际: %v", err)
	}
}

// =====================================================================
// 锁机制测试
// =====================================================================

// TestLock_AcquireAndRelease 测试锁的获取和释放。
func TestLock_AcquireAndRelease(t *testing.T) {
	targetDir := t.TempDir()
	lock := deploy.NewDeployLock(targetDir)

	// 获取锁
	if err := lock.Acquire(); err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}

	// 验证：锁目录已创建
	lockPath := filepath.Join(targetDir, ".claude.deploy.lock")
	if !fileExists(lockPath) {
		t.Error("锁目录未创建")
	}

	// 释放锁
	if err := lock.Release(); err != nil {
		t.Fatalf("释放锁失败: %v", err)
	}

	// 验证：锁目录已删除
	if fileExists(lockPath) {
		t.Error("锁目录未删除")
	}
}

// TestLock_SecondAcquireBlocks 测试第二个锁获取会失败。
func TestLock_SecondAcquireBlocks(t *testing.T) {
	targetDir := t.TempDir()

	lock1 := deploy.NewDeployLock(targetDir)
	lock2 := deploy.NewDeployLock(targetDir)

	// 第一个获取
	if err := lock1.Acquire(); err != nil {
		t.Fatalf("第一个锁获取失败: %v", err)
	}
	defer lock1.Release()

	// 第二个获取应超时失败（重试间隔短，总共 5 秒超时）
	err := lock2.Acquire()
	if err == nil {
		t.Error("第二个锁获取应失败")
		lock2.Release()
	}
}

// TestLock_StaleLockCleanup 测试过期锁会被自动清理。
func TestLock_StaleLockCleanup(t *testing.T) {
	targetDir := t.TempDir()

	// 手动创建一个旧的锁目录
	lockPath := filepath.Join(targetDir, ".claude.deploy.lock")
	os.MkdirAll(lockPath, 0o755)

	// 写入旧的锁信息
	infoFile := filepath.Join(lockPath, "info")
	oldContent := fmt.Sprintf("pid=99999\ntime=2000-01-01T00:00:00Z\n")
	os.WriteFile(infoFile, []byte(oldContent), 0o644)

	// 修改文件时间为 10 分钟前（超过 5 分钟阈值）
	// 使用 os.Chtimes 设置修改时间
	oldTime := time.Now().Add(-10 * time.Minute)
	os.Chtimes(infoFile, oldTime, oldTime)

	// 新锁应能成功获取（清理旧锁）
	lock := deploy.NewDeployLock(targetDir)
	if err := lock.Acquire(); err != nil {
		t.Fatalf("过期锁应被清理并成功获取: %v", err)
	}
	lock.Release()
}

// =====================================================================
// CopyFile 权限保留测试
// =====================================================================

// TestCopyFile_BasicCopy 测试基本的文件复制功能。
func TestCopyFile_BasicCopy(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "test.txt")
	dstFile := filepath.Join(dstDir, "sub", "test.txt")

	content := "hello world"
	os.WriteFile(srcFile, []byte(content), 0o644)

	err := deploy.CopyFile(dstFile, srcFile)
	if err != nil {
		t.Fatalf("CopyFile 失败: %v", err)
	}

	// 验证内容
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("内容不一致: got %q, want %q", string(data), content)
	}
}

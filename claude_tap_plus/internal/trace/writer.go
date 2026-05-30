package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/usage"
)

// TraceWriter 负责将 API 调用记录以 JSONL 格式追加写入追踪文件，
// 并在写入过程中累计 Token 用量统计。
//
// 使用方式：
//
//	tw, err := trace.NewTraceWriter(path)
//	tw.Write(record)
//	summary := tw.Summary()
//	tw.Close()
type TraceWriter struct {
	mu           sync.Mutex     // 保护并发写入和统计更新
	file         *os.File       // 底层追踪文件句柄
	writer       *bufio.Writer  // 缓冲写入器，提升写入性能
	count        int            // 已写入记录数（API 调用次数）
	inputTokens  int64          // 累计输入 Token
	outputTokens int64          // 累计输出 Token
	cacheRead    int64          // 累计缓存读取 Token
	cacheCreate  int64          // 累计缓存创建 Token
	modelsUsed   map[string]int // 各模型使用次数统计
	path         string         // 追踪文件路径
	sessionID    string         // 从首条记录提取的 session_id
}

// NewTraceWriter 创建 TraceWriter 实例，按需创建输出目录并打开追踪文件。
//
// 参数 path 为追踪文件的完整路径。函数会自动创建路径中缺失的目录，
// 并以追加模式打开文件（如果不存在则创建）。
func NewTraceWriter(path string) (*TraceWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create trace dir: %w", err)
	}
	logger.Debug("trace", "ensured directory: %s", dir)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open trace file: %w", err)
	}

	logger.Info("trace", "trace writer opened: %s", path)

	return &TraceWriter{
		file:       f,
		writer:     bufio.NewWriter(f),
		modelsUsed: make(map[string]int),
		path:       path,
	}, nil
}

// Write 将单条记录序列化为 JSON 行写入文件，并立即刷新缓冲区，
// 同时更新内部统计计数。
//
// 记录格式为 JSON Lines（每行一条独立 JSON，以换行符分隔），
// 便于后续按行读取和解析。
func (w *TraceWriter) Write(record map[string]any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal trace record: %w", err)
	}

	if _, err := w.writer.Write(data); err != nil {
		return err
	}
	if _, err := w.writer.WriteString("\n"); err != nil {
		return err
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}

	logger.Debug("trace", "record #%d written: %d bytes", w.count+1, len(data))

	w.count++
	w.updateStats(record)

	// 从首条记录提取 session_id
	if w.count == 1 {
		if sid, ok := record["session_id"].(string); ok {
			w.sessionID = sid
		}
	}
	return nil
}

// Close 刷新缓冲区并关闭底层文件。
// 应在追踪会话结束时调用，以确保所有数据落盘。
func (w *TraceWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	logger.Info("trace", "trace writer closed: %d records written", w.count)
	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Path 返回追踪文件的完整路径。
func (w *TraceWriter) Path() string {
	return w.path
}

// SessionID 返回从追踪记录中提取的 session_id。
// 如果尚无记录或首条记录不含 session_id，返回空字符串。
func (w *TraceWriter) SessionID() string {
	return w.sessionID
}

// Summary 返回当前追踪会话的聚合统计信息。
//
// 返回的 map 包含以下字段：
//   - api_calls:            API 调用总次数
//   - input_tokens:         累计输入 Token
//   - output_tokens:        累计输出 Token
//   - cache_read_tokens:    累计缓存读取 Token
//   - cache_create_tokens:  累计缓存创建 Token
//   - models_used:          各模型调用次数 map[string]int
func (w *TraceWriter) Summary() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()

	return map[string]any{
		"api_calls":           w.count,
		"input_tokens":        w.inputTokens,
		"output_tokens":       w.outputTokens,
		"cache_read_tokens":   w.cacheRead,
		"cache_create_tokens": w.cacheCreate,
		"models_used":         w.modelsUsed,
	}
}

// updateStats 从单条记录中提取模型信息和 Token 用量，更新累计统计。
//
// 提取逻辑：
//  1. 从 request.body.model 获取模型名称，归入 modelsUsed 计数
//  2. 从 response.body.usage 或 body 顶层提取 usage 数据
//  3. 使用 usage.NormalizeUsage 标准化后累加到各 Token 计数器
func (w *TraceWriter) updateStats(record map[string]any) {
	// 从请求体中提取模型名称
	reqBody, _ := record["request"].(map[string]any)
	if reqBody != nil {
		if body, ok := reqBody["body"].(map[string]any); ok {
			model, _ := body["model"].(string)
			if model == "" {
				model = "unknown"
			}
			w.modelsUsed[model]++
		}
	}

	// 从响应体中提取 Token 用量
	respBody, _ := record["response"].(map[string]any)
	if respBody == nil {
		return
	}
	body, _ := respBody["body"].(map[string]any)
	if body == nil {
		return
	}

	rawUsage, _ := body["usage"].(map[string]any)
	if rawUsage == nil {
		// 某些情况下 usage 直接位于 body 顶层
		rawUsage = body
	}

	norm := usage.NormalizeUsage(rawUsage)
	w.inputTokens += norm["input_tokens"]
	w.outputTokens += norm["output_tokens"]
	w.cacheRead += norm["cache_read_input_tokens"]
	w.cacheCreate += norm["cache_creation_input_tokens"]
}

// DefaultTraceDir 返回默认的追踪文件存储目录，位于用户主目录下。
//
// 默认路径：$HOME/.claude-tap-plus/.traces
// 如果无法获取用户主目录，则返回相对路径 ".claude-tap-plus/.traces"。
func DefaultTraceDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude-tap-plus/.traces"
	}
	dir := filepath.Join(home, ".claude-tap-plus", ".traces")
	logger.Debug("trace", "default trace dir: %s", dir)
	return dir
}

// DetectProjectName 返回当前工作目录对应的项目名称。
//
// 优先级：
//  1. 从 git remote origin URL 提取仓库名（如 my-project）
//  2. 当前工作目录的目录名作为兜底
func DetectProjectName() string {
	cwd, _ := os.Getwd()
	// 优先尝试从 git 远程地址获取仓库名
	if cwd != "" {
		repo := gitRemoteRepoName(cwd)
		if repo != "" {
			logger.Debug("trace", "detected project name: %s (from git)", repo)
			return repo
		}
	}
	// 兜底：使用当前目录名
	project := filepath.Base(cwd)
	if project == "" || project == "." {
		project = "default"
	}
	logger.Debug("trace", "detected project name: %s (from cwd)", project)
	return project
}

// gitRemoteRepoName 从 .git/config 文件中提取 remote origin 对应的仓库名。
//
// 示例：
//
//	"https://github.com/user/my-project.git" → "my-project"
//
// 直接读取配置文件以避免依赖 git 命令。
func gitRemoteRepoName(dir string) string {
	configPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	content := string(data)
	// 定位 [remote "origin"] 区域并提取 URL
	lines := strings.Split(content, "\n")
	inOrigin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin && strings.HasPrefix(trimmed, "url = ") {
			url := strings.TrimPrefix(trimmed, "url = ")
			// 提取 URL 最后一段并去掉 .git 后缀
			base := filepath.Base(url)
			base = strings.TrimSuffix(base, ".git")
			return base
		}
		if inOrigin && strings.HasPrefix(trimmed, "[") {
			break // 已进入下一个配置区域
		}
	}
	return ""
}

// MachineID 返回 "用户名@主机名" 格式的机器标识符。
// 用于区分不同机器的追踪数据。
func MachineID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	username := "unknown"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = u.Username
	}
	return username + "@" + hostname
}

// ExtractProjectSlug 从 Claude Code 的会话转录文件路径中提取项目标识（slug）。
//
// 输入示例：
//
//	"C:\Users\Admin\.claude\projects\D--xxx-yyy\uuid.jsonl"
//
// 输出：
//
//	"D--xxx-yyy"
//
// 如果路径不符合预期的 .claude/projects/ 模式，则返回空字符串。
func ExtractProjectSlug(transcriptPath string) string {
	if transcriptPath == "" {
		return ""
	}
	// 统一路径分隔符为 /
	normalized := filepath.ToSlash(transcriptPath)

	// 定位 ".claude/projects/" 标记段
	marker := ".claude/projects/"
	idx := strings.Index(normalized, marker)
	if idx < 0 {
		return ""
	}

	// 提取标记后的第一段目录名
	rest := normalized[idx+len(marker):]
	if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
		rest = rest[:slashIdx]
	}

	if rest == "" {
		return ""
	}
	return rest
}

// NewSessionTracePath 基于会话信息构建追踪文件路径。
//
// 路径格式：baseDir/{machineID}/{projectSlug}/{sessionID}.jsonl
//
// 同一 session_id 对应唯一路径，resume 时追加写入而非创建新文件。
//
// 参数说明：
//   - baseDir:     基础追踪目录
//   - machineID:   机器标识（如 username@hostname）
//   - projectSlug: 项目标识（如 D--xxx-yyy）
//   - sessionID:   会话唯一标识
func NewSessionTracePath(baseDir, machineID, projectSlug, sessionID string) string {
	path := filepath.Join(baseDir, machineID, projectSlug, sessionID+".jsonl")
	logger.Debug("trace", "session trace path: %s", path)
	return path
}

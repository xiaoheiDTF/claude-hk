// Package session 提供 Claude Code 会话的收集、恢复、状态查看等核心功能。
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionMeta 是每个项目存储在 meta.json 中的元数据结构。
type SessionMeta struct {
	Project   string         `json:"project"`             // 项目名称
	GitRemote string         `json:"git_remote,omitempty"` // Git 远程仓库地址（可选）
	LocalSlug string         `json:"local_slug"`          // 本地 slug（路径派生，唯一标识）
	LocalCwd  string         `json:"local_cwd"`           // 本地工作目录
	MachineID string         `json:"machine_id,omitempty"` // 机器 ID（可选）
	Sessions  []SessionEntry `json:"sessions"`            // 该项目下的会话列表
}

// SessionEntry 描述单个会话文件的信息。
type SessionEntry struct {
	SessionID      string   `json:"session_id"`      // 会话唯一 ID
	File           string   `json:"file"`            // 文件名
	FileSize       int64    `json:"file_size"`       // 文件大小（字节）
	RecordCount    int      `json:"record_count"`    // 记录条数
	FirstTimestamp string   `json:"first_timestamp"` // 第一条记录的时间戳
	LastTimestamp  string   `json:"last_timestamp"`  // 最后一条记录的时间戳
	ModelsUsed     []string `json:"models_used"`     // 使用的模型列表
	GitBranch      string   `json:"git_branch,omitempty"` // Git 分支（可选）
	SourceSlug     string   `json:"source_slug"`     // 来源 slug
	CollectedAt    string   `json:"collected_at"`    // 收集时间（RFC3339 格式）
}

// SessionDir 返回指定 slug 在 baseDir 下的会话存储目录。
// 使用 slug（由路径派生，全局唯一）作为目录名，可避免不同路径下同名的项目发生冲突
//（例如 D:\work\app 与 D:\personal\app）。
func SessionDir(baseDir, slug string) string {
	return filepath.Join(baseDir, "sessions", slug)
}

// BaseDir 返回可执行文件所在的目录，作为所有本地存储的根目录。
func BaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// LoadMeta 从指定目录读取 meta.json 文件。
// 如果文件不存在，则返回一个空的 SessionMeta（含空会话列表）。
func LoadMeta(dir string) (*SessionMeta, error) {
	path := filepath.Join(dir, "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionMeta{Sessions: []SessionEntry{}}, nil
		}
		return nil, fmt.Errorf("read meta.json: %w", err)
	}
	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse meta.json: %w", err)
	}
	return &meta, nil
}

// SaveMeta 将 meta.json 写入指定目录。
// 如果目录不存在会自动创建。
func SaveMeta(dir string, meta *SessionMeta) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta.json: %w", err)
	}
	path := filepath.Join(dir, "meta.json")
	return os.WriteFile(path, data, 0o644)
}

// ParseSessionJSONL 扫描一个 JSONL 文件以提取会话元数据。
// 逐行读取文件，收集：会话 ID、时间戳、使用的模型、Git 分支、记录条数等。
func ParseSessionJSONL(path string, sourceSlug string) (*SessionEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open jsonl: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat jsonl: %w", err)
	}

	scanner := bufio.NewScanner(f)
	// 允许每行最大 1MB（Claude 的 JSONL 行可能很大）。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		count     int
		sessionID string
		firstTS   string
		lastTS    string
		modelsMap = map[string]bool{}
		gitBranch string
	)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		count++

		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}

		// 从第一条记录中提取 session_id。
		if sessionID == "" {
			if sid, ok := record["sessionId"].(string); ok {
				sessionID = sid
			}
		}

		// 追踪最早和最晚的时间戳。
		if ts, ok := record["timestamp"].(string); ok && ts != "" {
			if firstTS == "" {
				firstTS = ts
			}
			lastTS = ts
		}

		// 从 message 字段中提取模型信息（助手回复记录）。
		if msg, ok := record["message"].(map[string]any); ok {
			if model, ok := msg["model"].(string); ok && model != "" {
				modelsMap[model] = true
			}
		}

		// 从用户记录中提取 Git 分支。
		if branch, ok := record["gitBranch"].(string); ok && branch != "" {
			gitBranch = branch
		}
	}

	// 如果 JSONL 中未提取到 session_id，则使用文件名作为回退。
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}

	models := make([]string, 0, len(modelsMap))
	for m := range modelsMap {
		models = append(models, m)
	}

	return &SessionEntry{
		SessionID:      sessionID,
		File:           filepath.Base(path),
		FileSize:       fi.Size(),
		RecordCount:    count,
		FirstTimestamp: firstTS,
		LastTimestamp:  lastTS,
		ModelsUsed:     models,
		GitBranch:      gitBranch,
		SourceSlug:     sourceSlug,
		CollectedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// FindSessionJSONLFiles 列出目录中所有看起来像会话文件的 .jsonl 文件。
// 会话文件名需符合 UUID 模式：{sessionId}.jsonl（至少包含 4 个短横线）。
func FindSessionJSONLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		// 会话文件格式为 {uuid}.jsonl — UUID 包含 4 个短横线。
		if strings.HasSuffix(name, ".jsonl") && strings.Count(name, "-") >= 4 {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files, nil
}

// copyFile 将单个文件从 src 复制到 dst。
// 会自动创建目标文件所在目录。
func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

package trace

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/usage"
)

// TraceWriter appends JSONL records to a trace file and accumulates token statistics.
type TraceWriter struct {
	mu           sync.Mutex
	file         *os.File
	writer       *bufio.Writer
	count        int
	inputTokens  int64
	outputTokens int64
	cacheRead    int64
	cacheCreate  int64
	modelsUsed   map[string]int
}

// NewTraceWriter creates the output directory if needed and opens the trace file for appending.
func NewTraceWriter(path string) (*TraceWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create trace dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open trace file: %w", err)
	}

	return &TraceWriter{
		file:       f,
		writer:     bufio.NewWriter(f),
		modelsUsed: make(map[string]int),
	}, nil
}

// Write serialises the record as a single JSON line, flushes, and updates statistics.
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

	w.count++
	w.updateStats(record)
	return nil
}

// Close flushes and closes the underlying file.
func (w *TraceWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Summary returns aggregate statistics for the trace session.
func (w *TraceWriter) Summary() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()

	return map[string]any{
		"api_calls":             w.count,
		"input_tokens":          w.inputTokens,
		"output_tokens":         w.outputTokens,
		"cache_read_tokens":     w.cacheRead,
		"cache_create_tokens":   w.cacheCreate,
		"models_used":           w.modelsUsed,
	}
}

func (w *TraceWriter) updateStats(record map[string]any) {
	// Extract model from request body.
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

	// Extract token usage from response body.
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
		// Sometimes the body itself is the usage object.
		rawUsage = body
	}

	norm := usage.NormalizeUsage(rawUsage)
	w.inputTokens += norm["input_tokens"]
	w.outputTokens += norm["output_tokens"]
	w.cacheRead += norm["cache_read_input_tokens"]
	w.cacheCreate += norm["cache_creation_input_tokens"]
}

// DefaultTraceDir returns the trace storage directory next to the executable.
// e.g. executable at "C:\tools\claude-tap-plus.exe" → "C:\tools\.traces"
func DefaultTraceDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ".traces"
	}
	return filepath.Join(filepath.Dir(exe), ".traces")
}

// DetectProjectName returns the project name for the current working directory.
// Priority: git remote repo name → cwd basename.
func DetectProjectName() string {
	cwd, _ := os.Getwd()
	// Try git remote origin URL first.
	if cwd != "" {
		repo := gitRemoteRepoName(cwd)
		if repo != "" {
			return repo
		}
	}
	// Fallback to cwd basename.
	project := filepath.Base(cwd)
	if project == "" || project == "." {
		project = "default"
	}
	return project
}

// NewTracePath returns a trace file path under baseDir/{project}/:
// baseDir/{project}/{date}_{time}_{shortID}.jsonl
// project is derived from git remote or cwd basename.
// shortID is 6 random hex chars for uniqueness (cloud-sync friendly).
func NewTracePath(baseDir string) string {
	project := DetectProjectName()

	now := time.Now()
	dateTime := now.Format("2006-01-02_150405")
	shortID := randomHex(3) // 3 bytes = 6 hex chars
	filename := fmt.Sprintf("%s_%s.jsonl", dateTime, shortID)

	return filepath.Join(baseDir, project, filename)
}

// gitRemoteRepoName extracts the repo name from git remote origin URL.
// e.g. "https://github.com/user/my-project.git" → "my-project"
func gitRemoteRepoName(dir string) string {
	// Try reading .git/config directly to avoid shell dependency.
	configPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	content := string(data)
	// Find [remote "origin"] section and extract URL.
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
			// Extract last segment, strip .git suffix.
			base := filepath.Base(url)
			base = strings.TrimSuffix(base, ".git")
			return base
		}
		if inOrigin && strings.HasPrefix(trimmed, "[") {
			break // entered next section
		}
	}
	return ""
}

// randomHex returns n random bytes as a hex string (2*n chars).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

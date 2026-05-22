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

// SessionMeta is the per-project metadata stored in meta.json.
type SessionMeta struct {
	Project   string         `json:"project"`
	GitRemote string         `json:"git_remote,omitempty"`
	LocalSlug string         `json:"local_slug"`
	LocalCwd  string         `json:"local_cwd"`
	MachineID string         `json:"machine_id,omitempty"`
	Sessions  []SessionEntry `json:"sessions"`
}

// SessionEntry describes a single session file.
type SessionEntry struct {
	SessionID     string   `json:"session_id"`
	File          string   `json:"file"`
	FileSize      int64    `json:"file_size"`
	RecordCount   int      `json:"record_count"`
	FirstTimestamp string   `json:"first_timestamp"`
	LastTimestamp  string   `json:"last_timestamp"`
	ModelsUsed    []string `json:"models_used"`
	GitBranch     string   `json:"git_branch,omitempty"`
	SourceSlug    string   `json:"source_slug"`
	CollectedAt   string   `json:"collected_at"`
}

// SessionDir returns the session storage directory for a slug under baseDir.
// Uses slug (path-derived, unique) to avoid collisions when multiple projects
// share the same basename at different paths (e.g. D:\work\app vs D:\personal\app).
func SessionDir(baseDir, slug string) string {
	return filepath.Join(baseDir, "sessions", slug)
}

// BaseDir returns the executable's directory (the root for all storage).
func BaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// LoadMeta reads meta.json from the given directory. Returns empty meta if not found.
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

// SaveMeta writes meta.json to the given directory.
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

// ParseSessionJSONL scans a JSONL file to extract session metadata.
// It reads the file line-by-line to collect: session ID, timestamps, models, branch, record count.
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
	// Allow lines up to 1MB (Claude JSONL lines can be large).
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

		// Extract session_id from first record.
		if sessionID == "" {
			if sid, ok := record["sessionId"].(string); ok {
				sessionID = sid
			}
		}

		// Track timestamps.
		if ts, ok := record["timestamp"].(string); ok && ts != "" {
			if firstTS == "" {
				firstTS = ts
			}
			lastTS = ts
		}

		// Extract model from message field (assistant records).
		if msg, ok := record["message"].(map[string]any); ok {
			if model, ok := msg["model"].(string); ok && model != "" {
				modelsMap[model] = true
			}
		}

		// Extract gitBranch from user records.
		if branch, ok := record["gitBranch"].(string); ok && branch != "" {
			gitBranch = branch
		}
	}

	// If no session_id from JSONL, try filename.
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}

	models := make([]string, 0, len(modelsMap))
	for m := range modelsMap {
		models = append(models, m)
	}

	return &SessionEntry{
		SessionID:     sessionID,
		File:          filepath.Base(path),
		FileSize:      fi.Size(),
		RecordCount:   count,
		FirstTimestamp: firstTS,
		LastTimestamp:  lastTS,
		ModelsUsed:    models,
		GitBranch:     gitBranch,
		SourceSlug:    sourceSlug,
		CollectedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// FindSessionJSONLFiles lists all .jsonl files in a directory that look like session files
// (filename is a UUID pattern: {sessionId}.jsonl).
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
		// Session files are {uuid}.jsonl — UUIDs have 4 dashes.
		if strings.HasSuffix(name, ".jsonl") && strings.Count(name, "-") >= 4 {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files, nil
}

// copyFile copies a single file from src to dst.
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

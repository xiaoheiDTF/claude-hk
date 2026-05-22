package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/session"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/testutil"
)

// TestGenerateSlug verifies slug generation for various path formats.
func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`D:\development\code\claude-tap`, "D--development-code-claude-tap"},
		{"/home/user/claude-tap", "-home-user-claude-tap"},
		{`C:\Users\EDY`, "C--Users-EDY"},
		{"/", "-"},
		{`D:\code\saas-rd\tripmax\database`, "D--code-saas-rd-tripmax-database"},
	}

	for _, tt := range tests {
		got := session.GenerateSlug(tt.input)
		if got != tt.want {
			t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCwdToClaudeKey verifies forward-slash conversion for .claude.json keys.
func TestCwdToClaudeKey(t *testing.T) {
	got := session.CwdToClaudeKey(`D:\development\code\claude-tap`)
	want := "D:/development/code/claude-tap"
	if got != want {
		t.Errorf("CwdToClaudeKey() = %q, want %q", got, want)
	}
}

// TestParseSessionJSONL uses the real session file from testdata.
func TestParseSessionJSONL(t *testing.T) {
	sessionDir := testutil.FindDir(t, filepath.Join("testdata", "sessions", "claude-tap"))
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		t.Skip("testdata/sessions/claude-tap not available")
	}

	jsonlCount := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		jsonlCount++
		path := filepath.Join(sessionDir, e.Name())
		se, err := session.ParseSessionJSONL(path, "D--development-code-claude-tap")
		if err != nil {
			t.Errorf("ParseSessionJSONL(%s): %v", path, err)
			continue
		}

		// Verify basic fields.
		if se.SessionID == "" {
			t.Errorf("%s: empty session_id", path)
		}
		if se.RecordCount == 0 {
			t.Errorf("%s: zero record_count", path)
		}
		if se.FileSize == 0 {
			t.Errorf("%s: zero file_size", path)
		}
		if se.FirstTimestamp == "" {
			t.Errorf("%s: empty first_timestamp", path)
		}
		if se.LastTimestamp == "" {
			t.Errorf("%s: empty last_timestamp", path)
		}
		if len(se.ModelsUsed) == 0 {
			t.Errorf("%s: empty models_used", path)
		}
		if se.SourceSlug != "D--development-code-claude-tap" {
			t.Errorf("%s: source_slug = %q, want D--development-code-claude-tap", path, se.SourceSlug)
		}

		t.Logf("✓ %s: %d records, %s, models=%v, branch=%s",
			se.SessionID[:8]+"...",
			se.RecordCount,
			formatSize(se.FileSize),
			se.ModelsUsed,
			se.GitBranch,
		)
	}

	if jsonlCount == 0 {
		t.Skip("no JSONL files in testdata/sessions/claude-tap/")
	}
}

// TestFindSessionJSONLFiles tests the file filtering logic.
func TestFindSessionJSONLFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files with different patterns.
	files := []string{
		"731bcd6d-b7eb-401f-a9f9-4ed73ce85b38.jsonl", // UUID format: 4+ dashes ✓
		"4a430bc7-cdd0-4010-adf2-70a4ddaacd36.jsonl", // UUID format: 4+ dashes ✓
		"simple.jsonl",                                // Too few dashes ✗
		"two-parts.jsonl",                             // Only 1 dash ✗
		"readme.md",                                   // Not JSONL ✗
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	found, err := session.FindSessionJSONLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(found) != 2 {
		t.Errorf("FindSessionJSONLFiles: got %d files, want 2", len(found))
		for _, f := range found {
			t.Logf("  found: %s", filepath.Base(f))
		}
	}
}

// TestLoadSaveMeta tests meta.json round-trip.
func TestLoadSaveMeta(t *testing.T) {
	dir := t.TempDir()

	meta := &session.SessionMeta{
		Project:   "test-project",
		GitRemote: "https://github.com/test/test.git",
		LocalSlug: "D--test-project",
		LocalCwd:  `D:\test\project`,
		Sessions: []session.SessionEntry{
			{
				SessionID:     "abc-123-def",
				File:          "abc-123-def.jsonl",
				FileSize:      1024,
				RecordCount:   50,
				FirstTimestamp: "2026-01-01T00:00:00Z",
				LastTimestamp:  "2026-01-01T01:00:00Z",
				ModelsUsed:    []string{"claude-sonnet-4-6"},
				GitBranch:     "main",
				SourceSlug:    "D--test-project",
				CollectedAt:   "2026-01-01T02:00:00Z",
			},
		},
	}

	if err := session.SaveMeta(dir, meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	loaded, err := session.LoadMeta(dir)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}

	if loaded.Project != meta.Project {
		t.Errorf("project: got %q, want %q", loaded.Project, meta.Project)
	}
	if loaded.GitRemote != meta.GitRemote {
		t.Errorf("git_remote: got %q, want %q", loaded.GitRemote, meta.GitRemote)
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("sessions: got %d, want 1", len(loaded.Sessions))
	}
	s := loaded.Sessions[0]
	if s.SessionID != "abc-123-def" {
		t.Errorf("session_id: got %q, want %q", s.SessionID, "abc-123-def")
	}
	if s.RecordCount != 50 {
		t.Errorf("record_count: got %d, want 50", s.RecordCount)
	}
}

// TestLoadMetaNotExist tests that LoadMeta returns empty meta for missing file.
func TestLoadMetaNotExist(t *testing.T) {
	meta, err := session.LoadMeta(t.TempDir())
	if err != nil {
		t.Fatalf("LoadMeta on empty dir: %v", err)
	}
	if meta.Project != "" {
		t.Errorf("expected empty project, got %q", meta.Project)
	}
	if len(meta.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(meta.Sessions))
	}
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return "big"
	case bytes >= KB:
		return "medium"
	default:
		return "small"
	}
}

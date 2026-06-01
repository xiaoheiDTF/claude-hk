// Package service 提供业务逻辑层实现。
package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// LogEntry 表示解析后的日志条目。
type LogEntry struct {
	Timestamp string // 完整时间戳，如 "2026-06-01 20:56:28.000"
	Level     string // 日志级别：DEBUG/INFO/WARN/ERROR
	Source    string // 来源模块
	Message   string // 日志内容
}

// LogService 提供日志查询功能，读取 logger.go 生成的日志文件。
type LogService struct {
	logDir string // 日志文件存放目录
}

// NewLogService 创建日志查询服务实例。
func NewLogService(logDir string) *LogService {
	return &LogService{logDir: logDir}
}

// QueryLogs 查询日志，支持按级别过滤、按日期选择、限制返回条数。
func (svc *LogService) QueryLogs(_ context.Context, level, date string, limit int) ([]LogEntry, error) {
	logger.Debug("svc.log", "QueryLogs: level=%s date=%s limit=%d", level, date, limit)

	if limit <= 0 {
		limit = 100
	}

	// 确定日志文件日期
	logDate := date
	if logDate == "" {
		logDate = time.Now().Format("2006-01-02")
	}

	logFile := filepath.Join(svc.logDir, logDate+".log")

	// 文件不存在则返回空数组
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	entries, err := svc.parseLogFile(logFile, logDate, level, limit)
	if err != nil {
		return nil, fmt.Errorf("parse log file: %w", err)
	}

	return entries, nil
}

// logPattern 匹配 logger.go 的日志行格式：
//
//	15:04:05.000 [LEVEL] module: message
//
// 例如：20:56:28.000 [INFO ] backend.cmd: backend starting: host=127.0.0.1
var logPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2}\.\d{3}) \[([^\]]+)\]\s*(\S+): (.*)$`)

// parseLogFile 逐行解析日志文件，按级别过滤并限制条数。
func (svc *LogService) parseLogFile(path, logDate, levelFilter string, limit int) ([]LogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() && len(entries) < limit {
		line := scanner.Text()
		matches := logPattern.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		entry := LogEntry{
			Timestamp: logDate + " " + matches[1], // 补全日期
			Level:     strings.TrimSpace(matches[2]),
			Source:    matches[3],
			Message:   matches[4],
		}

		// 级别过滤（大小写不敏感）
		if levelFilter != "" && !strings.EqualFold(entry.Level, levelFilter) {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

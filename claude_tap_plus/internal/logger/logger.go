// Package logger 提供按日期分割的文件日志功能。
//
// 日志文件命名格式：{dir}/YYYY-MM-DD.log
// 日志行格式：15:04:05.000 [LEVEL] module: message
//
// 使用方式：
//
//	logger.Init(dir, true, logger.DEBUG)
//	defer logger.Close()
//	logger.Info("proxy", "started on :8080")
//	logger.Debug("store", "query: %s", sql)
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level 表示日志级别。
type Level int

const (
	DEBUG Level = iota // DEBUG 调试级别
	INFO               // INFO  信息级别
	WARN               // WARN  警告级别
	ERROR              // ERROR 错误级别
)

var (
	mu          sync.Mutex // 保护文件句柄与全局状态
	file        *os.File   // 当前日志文件
	dir         string     // 日志存放目录
	console     bool       // 是否同时输出到 stderr
	minLevel    Level = INFO
	currentDate string     // 当前日志文件对应的日期
	writer      io.Writer  // 包装 file 的写入器
)

// Init 初始化全局日志器。
//
// 参数：
//   - dir: 日志文件存放目录（自动创建）
//   - consoleEnabled: 是否同时输出到 stderr（仅 INFO 及以上）
//   - level: 最低日志级别
//
// 可多次调用以切换目录或级别。
func Init(d string, consoleEnabled bool, level Level) error {
	mu.Lock()
	defer mu.Unlock()

	minLevel = level
	console = consoleEnabled

	if d == "" {
		// 空目录：静默模式，所有写入被丢弃
		if file != nil {
			file.Close()
		}
		file = nil
		dir = ""
		currentDate = ""
		writer = nil
		return nil
	}

	// 创建目录
	if err := os.MkdirAll(d, 0o755); err != nil {
		return fmt.Errorf("logger: create dir %s: %w", d, err)
	}

	dir = d

	// 打开当天的日志文件
	today := time.Now().Format("2006-01-02")
	if err := rotateFile(today); err != nil {
		return err
	}

	return nil
}

// Close 刷新并关闭当前日志文件。
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		err := file.Close()
		file = nil
		writer = nil
		currentDate = ""
		return err
	}
	return nil
}

// SetLevel 运行时调整最低日志级别。
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	minLevel = level
}

// Debug 记录 DEBUG 级别日志。
func Debug(module, format string, args ...any) {
	write(DEBUG, module, format, args...)
}

// Info 记录 INFO 级别日志。
func Info(module, format string, args ...any) {
	write(INFO, module, format, args...)
}

// Warn 记录 WARN 级别日志。
func Warn(module, format string, args ...any) {
	write(WARN, module, format, args...)
}

// Error 记录 ERROR 级别日志。
func Error(module, format string, args ...any) {
	write(ERROR, module, format, args...)
}

var levelNames = [4]string{"DEBUG", "INFO ", "WARN ", "ERROR"}

// write 是所有日志方法的核心实现。
//
// 检查级别 → 加锁 → 检查日期轮转 → 格式化 → 写文件 → 可选写 stderr
func write(level Level, module, format string, args ...any) {
	if level < minLevel {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// 日期轮转检查（懒检测）
	today := time.Now().Format("2006-01-02")
	if today != currentDate && dir != "" {
		_ = rotateFile(today)
	}

	// 格式化日志行：15:04:05.000 [LEVEL] module: message
	ts := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", ts, levelNames[level], module, msg)

	// 写入文件
	if writer != nil {
		_, _ = writer.Write([]byte(line))
	}

	// 可选：同时输出到 stderr（仅 INFO 及以上）
	if console && level >= INFO {
		_, _ = fmt.Fprint(os.Stderr, line)
	}
}

// rotateFile 关闭旧文件并打开新日期的日志文件。
// 调用方必须持有 mu 锁。
func rotateFile(date string) error {
	if file != nil {
		file.Close()
		file = nil
		writer = nil
	}

	path := filepath.Join(dir, date+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to open %s: %v\n", path, err)
		return err
	}

	file = f
	writer = f
	currentDate = date
	return nil
}

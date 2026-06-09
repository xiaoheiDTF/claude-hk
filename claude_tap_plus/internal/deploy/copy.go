package deploy

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFile 复制单个文件，保留源文件权限。
func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// CopyDir 递归复制整个目录，保留文件权限。
func CopyDir(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return CopyFile(dstPath, path)
	})
}

// CopyDirFiltered 递归复制目录，只复制 shouldCopy 返回 true 的文件。
// 目录结构始终完整创建。
func CopyDirFiltered(dst, src string, shouldCopy func(relPath string) bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 使用统一的路径分隔符用于匹配
		normalizedRelPath := filepath.ToSlash(relPath)
		if !shouldCopy(normalizedRelPath) {
			return nil
		}

		return CopyFile(dstPath, path)
	})
}

package deploy

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrTargetNotExist 目标路径不存在。
	ErrTargetNotExist = errors.New("目标路径不存在")
	// ErrTargetNotDir 目标路径不是目录。
	ErrTargetNotDir = errors.New("目标路径不是目录")
)

// ValidateTarget 校验目标路径是否有效（存在且是目录）。
func ValidateTarget(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrTargetNotExist, path)
		}
		return fmt.Errorf("访问目标路径失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrTargetNotDir, path)
	}
	return nil
}

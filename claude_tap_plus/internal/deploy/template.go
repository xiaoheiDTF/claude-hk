package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// requiredItems 是模板目录必须包含的子项。
var requiredItems = []string{
	"hooks/",
	"skills/",
	"settings.json",
	"init.sh",
	"dirs.conf",
	"skills/registry.conf",
	"lib/",
	"scripts/",
	"myRule/",
}

// ExeDir 返回可执行文件所在目录。
func ExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	return filepath.Dir(exe), nil
}

// TemplateDir 返回模板目录路径（可执行文件同级的 template/ 目录）。
func TemplateDir() (string, error) {
	dir, err := ExeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "template"), nil
}

// ValidateTemplate 校验模板目录是否存在且包含所有必需项。
// 返回的错误信息包含具体的缺失项列表。
func ValidateTemplate(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("模板目录不存在: %s", dir)
		}
		return fmt.Errorf("访问模板目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("模板路径不是目录: %s", dir)
	}

	// 检查目录是否为空
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取模板目录失败: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("模板目录为空: %s", dir)
	}

	// 检查必需项
	var missing []string
	for _, item := range requiredItems {
		path := filepath.Join(dir, item)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, item)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("模板目录缺少必需项: %v", missing)
	}

	return nil
}

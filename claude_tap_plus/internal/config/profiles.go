package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Alias 表示一个别名：Claude Code 侧使用的 model 名（如 "opus[1m]"）绑定一组后端凭证。
// proxy 收到请求后，按请求体 model 字段（即别名 name）查表，改写为真实 model 并选用对应凭证转发。
type Alias struct {
	Name      string `json:"name"`                 // 必填，Claude Code 用作 model 名（可带 [1m] 等后缀）
	Model     string `json:"model"`                // 必填，转发时写入请求体 model 字段的真实模型名
	BaseURL   string `json:"base_url"`             // 必填，后端 API 地址
	APIKey    string `json:"api_key,omitempty"`    // 与 auth_token 互斥；同时配置时以 auth_token 为准
	AuthToken string `json:"auth_token,omitempty"` // OAuth token，与 api_key 互斥，优先级更高
	Provider  string `json:"provider,omitempty"`   // anthropic/openai/gemini，缺省 anthropic；决定 api_key 的鉴权头格式
	KimiMode  *bool  `json:"kimi_mode,omitempty"`  // 显式指定是否注入 reasoning_content；nil 时按 base_url 自动判断
}

// Profile 是一个启动配置，只承载注入给 Claude Code 子进程的 env 变量（客户端行为配置）。
// 后端凭证不在这里，凭证集中在 aliases 表全局共享。
type Profile struct {
	Env map[string]string `json:"env"`
}

// ProfilesFile 是 profiles.json 的完整结构。
//
//	default_profile — 未传 --tap-profile 时使用的 profile 名
//	default_alias   — 请求 model 未命中任何别名时的兜底别名
//	aliases         — 全局共享的 key 池 + 路由目标
//	profiles        — 启动配置（只含 env）
type ProfilesFile struct {
	DefaultProfile string             `json:"default_profile"`
	DefaultAlias   string             `json:"default_alias"`
	Aliases        []Alias            `json:"aliases"`
	Profiles       map[string]Profile `json:"profiles"`
}

// ConfigDir 返回 claude-tap-plus 的配置根目录（~/.claude-tap-plus/）。
func ConfigDir() string {
	return filepath.Join(HomeDir(), ".claude-tap-plus")
}

// profilesPath 返回 profiles.json 的完整路径。
func profilesPath() string {
	return filepath.Join(ConfigDir(), "profiles.json")
}

// ReadProfiles 读取并解析 ~/.claude-tap-plus/profiles.json（新格式）。
//
// 文件不存在：返回 (nil, nil)（非致命，调用方据此判定走 bypass 自动探测）。
// 旧扁平格式：返回 error（F4.2 检测到旧格式，打印迁移提示并退出）。
func ReadProfiles() (*ProfilesFile, error) {
	path := profilesPath()
	logger.Debug("config", "reading profiles: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("config", "profiles.json not found")
			return nil, nil
		}
		return nil, fmt.Errorf("read profiles: %w", err)
	}

	if detectOldFormat(data) {
		return nil, fmt.Errorf("检测到旧格式 profiles.json（扁平 ProfileConfig：base_url/api_key/model 等），与本期别名路由不兼容。\n请按新格式迁移（顶层 aliases + profiles.<name>.env），参考 PRD《别名路由与多环境隔离》第 4 节")
	}

	var pf ProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	pf.normalize()

	logger.Debug("config", "profiles loaded: default_profile=%s default_alias=%s aliases=%d profiles=%d",
		pf.DefaultProfile, pf.DefaultAlias, len(pf.Aliases), len(pf.Profiles))
	return &pf, nil
}

// detectOldFormat 探测 profiles.json 是否为旧扁平格式。
// 判定：任一 profiles[name] 直接携带 base_url/api_key/auth_token/model/provider 字段。
// 新格式 profile 形如 {"env": {...}}，反序列化到扁平探针结构时这些字段为空，不会误判。
func detectOldFormat(data []byte) bool {
	var probe struct {
		Profiles map[string]json.RawMessage `json:"profiles"`
	}
	if json.Unmarshal(data, &probe) != nil {
		return false
	}
	for _, raw := range probe.Profiles {
		var flat struct {
			BaseURL   string `json:"base_url"`
			APIKey    string `json:"api_key"`
			AuthToken string `json:"auth_token"`
			Model     string `json:"model"`
			Provider  string `json:"provider"`
		}
		if json.Unmarshal(raw, &flat) != nil {
			continue
		}
		if flat.BaseURL != "" || flat.APIKey != "" || flat.AuthToken != "" || flat.Model != "" || flat.Provider != "" {
			return true
		}
	}
	return false
}

// normalize 对解析结果做一次性规整：
//   - provider 缺省补 "anthropic"
//   - api_key 与 auth_token 同时配置：warn + 清空 api_key（以 auth_token 为准）
//
// 同名别名的"后者覆盖"由 aliasMap 在查询时自然实现，这里仅记录 warn。
func (pf *ProfilesFile) normalize() {
	seen := make(map[string]bool, len(pf.Aliases))
	for i := range pf.Aliases {
		a := &pf.Aliases[i]
		if a.Provider == "" {
			a.Provider = "anthropic"
		}
		if a.AuthToken != "" && a.APIKey != "" {
			logger.Warn("config", "别名 %q 同时配置 api_key 与 auth_token，以 auth_token 为准（忽略 api_key）", a.Name)
			a.APIKey = ""
		}
		if seen[a.Name] {
			logger.Warn("config", "别名 %q 重复，后者将覆盖前者", a.Name)
		}
		seen[a.Name] = true
	}
}

// aliasMap 构建 name→Alias 映射，遇到重复 name 时后者覆盖（决策 1）。
func (pf *ProfilesFile) aliasMap() map[string]Alias {
	m := make(map[string]Alias, len(pf.Aliases))
	for _, a := range pf.Aliases {
		m[a.Name] = a
	}
	return m
}

// ResolveAlias 按 name 精确匹配别名；未命中时使用 default_alias 兜底（F1.5）。
// 第二返回值表示是否命中（含 default_alias 兜底）。
func (pf *ProfilesFile) ResolveAlias(name string) (Alias, bool) {
	m := pf.aliasMap()
	if a, ok := m[name]; ok {
		return a, true
	}
	if pf.DefaultAlias != "" {
		if a, ok := m[pf.DefaultAlias]; ok {
			return a, true
		}
	}
	return Alias{}, false
}

// ResolveFallbackAliases 返回真实 model 与 realModel 相同、且 name 不等于 excludeName 的别名，按数组顺序（F2.2）。
// 本期不引入 priority，顺序即兜底顺序。
func (pf *ProfilesFile) ResolveFallbackAliases(realModel, excludeName string) []Alias {
	if realModel == "" {
		return nil
	}
	var out []Alias
	for _, a := range pf.Aliases {
		if a.Name == excludeName {
			continue
		}
		if a.Model == realModel {
			out = append(out, a)
		}
	}
	return out
}

// forbiddenProfileEnv 列出禁止出现在 profile.env 中的变量（与 proxy 运行契约冲突）。
var forbiddenProfileEnv = map[string]string{
	"ANTHROPIC_BASE_URL":      "proxy 已强制设为本地地址，不可在 profile.env 中覆盖",
	"CLAUDE_CODE_USE_BEDROCK": "代理场景不支持 Bedrock provider 开关",
	"CLAUDE_CODE_USE_VERTEX":  "代理场景不支持 Vertex provider 开关",
	"CLAUDE_CODE_USE_FOUNDRY": "代理场景不支持 Foundry provider 开关",
}

// ResolveProfileEnv 返回指定 profile 的 env，剔除与 proxy 冲突的禁止项（F3.3）。
// name 为空时使用 default_profile；均无则返回 error。
func (pf *ProfilesFile) ResolveProfileEnv(name string) (map[string]string, error) {
	if name == "" {
		name = pf.DefaultProfile
	}
	if name == "" {
		return nil, fmt.Errorf("no profile name specified and no default_profile configured")
	}
	p, ok := pf.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in profiles.json", name)
	}

	env := make(map[string]string, len(p.Env))
	for k, v := range p.Env {
		if reason, bad := forbiddenProfileEnv[strings.ToUpper(k)]; bad {
			logger.Warn("config", "profile.env 忽略禁止项 %s: %s", k, reason)
			continue
		}
		env[k] = v
	}
	return env, nil
}

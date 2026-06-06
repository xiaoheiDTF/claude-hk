// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ReasoningCache 缓存 API 响应中的 reasoning_content，用于在后续请求中回填。
//
// Kimi 等兼容 Anthropic 的端点要求 assistant tool_call 消息必须携带 reasoning_content 字段。
// Claude Code 在发送多轮对话时不会回传该字段，因此需要代理层缓存并回填。
//
// 缓存策略为三级查找（与 Python 参考实现一致）：
//  1. full key 精确匹配（content + tool_calls）
//  2. toolcall key 匹配（仅 tool_calls，适用于 content 为空的情况）
//  3. 最近一条缓存值兜底（仅对带 tool_calls 的消息生效）
type ReasoningCache struct {
	mu        sync.RWMutex
	full      map[string]string // full_key → reasoning_content
	toolcall  map[string]string // toolcall_key → reasoning_content
	lastValue string            // 最近缓存的值（兜底用）
}

// NewReasoningCache 创建一个空的缓存实例。
func NewReasoningCache() *ReasoningCache {
	return &ReasoningCache{
		full:     make(map[string]string),
		toolcall: make(map[string]string),
	}
}

// Store 将 reasoning_content 存入缓存。
//
// fullKey: 基于 content + tool_calls 计算的完整密钥。
// tcKey: 仅基于 tool_calls 计算的密钥（空字符串时跳过 toolcall 缓存）。
// reasoningContent: 要缓存的推理内容（空字符串时不缓存）。
func (c *ReasoningCache) Store(fullKey, tcKey, reasoningContent string) {
	if reasoningContent == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.full[fullKey] = reasoningContent
	if tcKey != "" {
		c.toolcall[tcKey] = reasoningContent
	}
	c.lastValue = reasoningContent
}

// Lookup 三级查找 reasoning_content。
//
// 返回 (reasoning_content, found)：
//   - found=true 表示命中缓存
//   - found=false 表示未命中，调用方应使用空字符串兜底
func (c *ReasoningCache) Lookup(fullKey, tcKey string, hasToolCalls bool) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Level 1: full key 精确匹配
	if v, ok := c.full[fullKey]; ok {
		return v, true
	}

	// Level 2: toolcall key 匹配
	if tcKey != "" {
		if v, ok := c.toolcall[tcKey]; ok {
			return v, true
		}
	}

	// Level 3: 兜底 - 最近缓存值（仅对带 tool_calls 的消息）
	if hasToolCalls && c.lastValue != "" {
		return c.lastValue, true
	}

	return "", false
}

// Len 返回缓存条目数 (fullCount, toolcallCount)。
func (c *ReasoningCache) Len() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.full), len(c.toolcall)
}

// ---- 密钥计算 ----

// makeCacheKey 计算缓存密钥：MD5(format)。
func makeCacheKey(format string) string {
	h := md5.Sum([]byte(format))
	return fmt.Sprintf("%x", h)
}

// MakeFullKey 基于 content 文本和 tool_calls JSON 计算完整密钥。
// 格式与 Python 参考实现一致：MD5("content=<content>\ntool_calls=<toolCallsJSON>")
func MakeFullKey(content string, toolCallsJSON string) string {
	return makeCacheKey(fmt.Sprintf("content=%s\ntool_calls=%s", content, toolCallsJSON))
}

// MakeToolcallKey 仅基于 tool_calls JSON 计算密钥。
// 如果 toolCallsJSON 为空，返回空字符串（表示无需缓存到 toolcall map）。
func MakeToolcallKey(toolCallsJSON string) string {
	if toolCallsJSON == "" {
		return ""
	}
	return makeCacheKey(fmt.Sprintf("tool_calls=%s", toolCallsJSON))
}

// MakeFullKeyFromMsg 从 Anthropic 格式的 assistant 消息中提取密钥。
//
// 遍历 content 数组，将非 thinking 块序列化为 content，
// 将 tool_use 块序列化为 tool_calls，最终调用 MakeFullKey。
func MakeFullKeyFromMsg(msg map[string]any) string {
	content, toolCalls := extractMsgParts(msg)
	toolCallsJSON, _ := json.Marshal(toolCalls)
	return MakeFullKey(content, string(toolCallsJSON))
}

// MakeToolcallKeyFromMsg 从 Anthropic 格式的 assistant 消息中提取 toolcall 密钥。
func MakeToolcallKeyFromMsg(msg map[string]any) string {
	_, toolCalls := extractMsgParts(msg)
	if len(toolCalls) == 0 {
		return ""
	}
	toolCallsJSON, _ := json.Marshal(toolCalls)
	return MakeToolcallKey(string(toolCallsJSON))
}

// extractMsgParts 从 Anthropic 格式消息中提取文本内容和 tool_use 块。
//
// 返回 (contentText, toolUseBlocks)：
//   - contentText: 所有 text 块拼接的文本内容
//   - toolUseBlocks: 所有 tool_use 块的数组（保持原始顺序）
func extractMsgParts(msg map[string]any) (string, []any) {
	contentArr, ok := msg["content"].([]any)
	if !ok {
		// content 可能是字符串
		if s, ok := msg["content"].(string); ok {
			return s, nil
		}
		return "", nil
	}

	var textParts []string
	var toolUseBlocks []any

	for _, blockRaw := range contentArr {
		block, ok := blockRaw.(map[string]any)
		if !ok {
			continue
		}
		blockType, _ := block["type"].(string)
		switch blockType {
		case "text":
			if text, ok := block["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case "tool_use":
			toolUseBlocks = append(toolUseBlocks, block)
		}
	}

	return strings.Join(textParts, ""), toolUseBlocks
}

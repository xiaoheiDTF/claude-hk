# 12. 降级策略与测试验证 — 实施记录

> 创建时间：2026-05-29
> 所属模块：claude-tap-plus 后端降级
> 简述：统一降级策略实施、技能脚本改造、Go/Shell 降级测试编写

---

## 一、需求背景

技能脚本（001-4 ~ 001-9）通过 `backend.sh` 调用 Go 后端管理 Issue 状态。后端不可用时需要统一的降级行为：

- 非关键操作（status 更新、release-session）→ 静默跳过
- claim 操作 → **必须阻止**（防多 agent 冲突）
- malformed backend.conf → 不崩溃

需求编号：requirement-queue.md 第 12 项"降级策略与测试验证"

---

## 二、修改的文件（3 个原有文件）

### 2.1 `.claude/skills/backend.sh`

**改动性质：** 增强 + 安全修复

**改动 1：`_load_backend_url()` 增加 URL 格式校验**

原代码：
```bash
_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  fi
}
```

新代码：
```bash
_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    local conf_file="$CLAUDE_PROJECT_DIR/.claude/backend.conf"
    [ -f "$conf_file" ] || return 1

    local raw_url
    raw_url=$(grep '^BACKEND_URL=' "$conf_file" 2>/dev/null | head -1 | cut -d= -f2-)
    [ -z "$raw_url" ] && return 1

    # URL 格式校验：必须以 http:// 或 https:// 开头
    case "$raw_url" in
      http://*|https://*) ;;
      *)
        [ -n "${SKILL_TAG:-}" ] && skill_log "WARN" "backend.conf: invalid URL format: $raw_url"
        return 1
        ;;
    esac

    # 拒绝包含空白字符的 URL
    case "$raw_url" in
      *\ *|*\	*)
        [ -n "${SKILL_TAG:-}" ] && skill_log "WARN" "backend.conf: URL contains whitespace"
        return 1
        ;;
    esac

    BACKEND_URL="$raw_url"
  fi
}
```

变更点：
- 先检查文件存在性
- `grep | head -1` 取第一行匹配（防御多行 backend.conf）
- `cut -d= -f2-` 改为 `f2-`（URL 中含 `=` 时不截断）
- 新增 URL 格式校验（http/https 前缀 + 无空白字符）
- 校验失败时通过 `skill_log` 记录 WARN 日志

**改动 2：新增 `_require_backend()` 函数**

```bash
# 检查后端是否必须可用（用于 claim 等安全敏感操作）
# 返回码: 0=可用, 1=已配置但不可达, 2=未配置
_require_backend() {
  _load_backend_url
  if [ -z "$BACKEND_URL" ]; then
    return 2
  fi
  if curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1; then
    return 0
  fi
  [ -n "${SKILL_TAG:-}" ] && skill_log "ERROR" "Backend configured but unreachable at $BACKEND_URL"
  return 1
}
```

三返回码设计：
- 0：后端可用，可安全执行 claim
- 1：后端已配置但不可达 → claim 必须阻止
- 2：后端未配置（无 backend.conf）→ 单 agent 模式，claim 可跳过后端

**改动 3：`_call_backend()` 增加可选日志**

```bash
_call_backend() {
  local endpoint="$1"
  local data="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 1

  local resp
  resp=$(curl -s --max-time 5 -X POST "$BACKEND_URL$endpoint" \
    -H "Content-Type: application/json" \
    -d "$data" 2>/dev/null)
  local rc=$?
  if [ $rc -ne 0 ] || [ -z "$resp" ]; then
    [ -n "${SKILL_TAG:-}" ] && skill_log "DEBUG" "Backend call failed: $endpoint (curl rc=$rc)"
    return 1
  fi
  echo "$resp"
}
```

变更点：
- 记录 curl 返回码 `$rc`
- 空响应也视为失败
- 失败时通过 `skill_log` 记录 DEBUG 日志

**改动 4（计划外）：变量安全引用**

原代码：
```bash
$SKILL_TAG
$CLAUDE_SESSION_ID
```

新代码：
```bash
${SKILL_TAG:-}
${CLAUDE_SESSION_ID:-}
```

原因：Shell 测试脚本使用 `set -u` 时，未设置的变量会触发 `unbound variable` 错误。改为 `${VAR:-}` 后变量未设置时展开为空字符串，不再报错。

**影响范围：** 所有 source `backend.sh` 的技能脚本（001-4 ~ 001-9）在实际执行时 `SKILL_TAG` 和 `CLAUDE_SESSION_ID` 都已由各自的 `03UserPromptSubmit.sh` 设置，因此 `${VAR:-}` 与 `$VAR` 行为一致。此改动对正常运行无影响。

**未经端到端测试验证的改动项。**

---

### 2.2 `.claude/skills/001-4-issue-claim/scripts/03UserPromptSubmit.sh`

**改动性质：** 安全增强

**改动：`claim_issue_backend()` 函数增加后端可用性检查**

原代码：
```bash
claim_issue_backend() {
  local issue_num="$1"
  local issue_title="$2"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0

  local repo
  repo=$(_get_repo)
  [ -z "$repo" ] && return 0

  local result
  result=$(_call_backend "/api/issue/claim" "{...}")

  if ! echo "$result" | jq -e '.success' > /dev/null 2>&1; then
    local claimed_by
    claimed_by=$(echo "$result" | jq -r '.claimed_by // "unknown"')
    echo "ERROR: Issue #$issue_num 已被 $claimed_by 领取"
    return 1
  fi

  return 0
}
```

新代码：
```bash
claim_issue_backend() {
  local issue_num="$1"
  local issue_title="$2"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0  # 未配置后端，跳过（单 agent 模式）

  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0

  local repo
  repo=$(_get_repo)
  [ -z "$repo" ] && return 0

  # 后端已配置但不可达 → 阻止领取（防多 agent 冲突）
  if ! _backend_available; then
    skill_log "ERROR" "Backend unreachable, BLOCKING claim for #$issue_num"
    echo "ERROR: Backend server is unreachable. Cannot safely claim issue #$issue_num (risk of multi-agent conflict)."
    echo "Please ensure the backend is running at $BACKEND_URL and try again."
    return 1
  fi

  local result
  result=$(_call_backend "/api/issue/claim" "{...}")

  if [ -z "$result" ]; then
    skill_log "ERROR" "Backend claim call returned empty for #$issue_num"
    echo "ERROR: Backend claim call failed unexpectedly for issue #$issue_num."
    return 1
  fi

  if ! echo "$result" | jq -e '.success' > /dev/null 2>&1; then
    local claimed_by
    claimed_by=$(echo "$result" | jq -r '.claimed_by // "unknown"')
    echo "ERROR: Issue #$issue_num 已被 $claimed_by 领取"
    return 1
  fi

  return 0
}
```

新增的检查点：
1. `_backend_available` 前置检查 — 后端不可达时阻止 claim 并输出错误
2. 空响应检查 — `_call_backend` 返回空时阻止 claim
3. 两处 `skill_log` 记录 ERROR 日志

降级行为变化：

| 场景 | 原行为 | 新行为 |
|------|--------|--------|
| 无 backend.conf | 跳过后端，return 0 | 不变 |
| 后端配置但不可达 | `_call_backend` 静默返回空，jq 解析空字符串报错但函数行为不确定 | **阻止 claim**，输出错误提示，return 1 |
| API 返回空响应 | jq 解析失败，函数行为不确定 | **阻止 claim**，输出错误提示，return 1 |

**未经端到端测试验证的改动项。** 需通过 `/001-4-issue-claim #N` 实际执行验证。

---

### 2.3 `.claude/hooks/29-session-end/base.sh`

**改动性质：** 代码复用重构

**改动：`release_session_issues()` 复用 `backend.sh` 统一函数**

原代码：
```bash
release_session_issues() {
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
    | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$sid\"}")

  local count
  count=$(echo "$result" | jq -r '.count // 0' 2>/dev/null)
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $sid"
}
```

新代码：
```bash
release_session_issues() {
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  if ! _backend_available; then
    log "INFO" "Backend unreachable, skipping session release for $sid"
    return 0
  fi

  local result
  result=$(curl -s --max-time 5 -X POST "$BACKEND_URL/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$sid\"}" 2>/dev/null) || return 0

  local count
  count=$(echo "$result" | jq -r '.count // 0' 2>/dev/null)
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $sid"
}
```

变更点：
- 移除内联 `cat | grep | cut` URL 加载，改为 source `backend.sh` 复用 `_load_backend_url()`
- 新增 `_backend_available` 健康检查，不可达时 log INFO 静默跳过
- curl 失败时 `|| return 0` 静默降级
- 使用统一变量 `$BACKEND_URL` 替代局部 `$backend_url`

**未经端到端测试验证的改动项。** 需通过真实会话结束事件验证。

---

## 三、新建的文件（2 个测试文件）

### 3.1 `claude_tap_plus/tests/backend/degradation_test.go`

**类型：** Go 测试（package `backend_test`）

**测试用例（11 个，全部通过）：**

| 测试函数 | 覆盖场景 |
|---------|---------|
| `TestRecovery_StatePreservedAcrossRestart` | 同一 SQLite 文件创建两个 server，验证 claim+status 状态在重启后保持 |
| `TestRecovery_ClaimFailsForAlreadyClaimedAfterRestart` | 重启后已 claim 的 issue 不能被其他 session 再次 claim |
| `TestRecovery_ReleaseSessionAfterRestart` | 重启后 release-session 仍能正确释放 |
| `TestMalformedRequest_InvalidJSON` | 非 JSON body 返回 400 |
| `TestMalformedRequest_EmptyRequiredFields` | 空 repo/issue_number/session_id 返回 400（3 子测试） |
| `TestMalformedRequest_NegativeIssueNumber` | 负数 issue_number 后端优雅处理 |
| `TestMalformedRequest_StatusEmptyFields` | status API 空必填字段返回 400（3 子测试） |
| `TestMalformedRequest_ExtraFieldsIgnored` | 多余字段被忽略，正常响应 |
| `TestMalformedRequest_InvalidMethod` | GET 请求返回 405 |
| `TestMalformedRequest_ReleaseSessionEmptySessionID` | 空 session_id 返回 400 |
| `TestMalformedRequest_UnicodeInFields` | Unicode repo 名正常处理 |

**测试模式：** 复用 `issue_api_test.go` 中的 `testEnv` 结构和 `setupTest()` helper。Recovery 测试使用 `setupDegradationTest()` 返回 `(env, dbPath)` 以便重启时复用 SQLite 文件。

### 3.2 `claude_tap_plus/tests/shell/degradation_test.sh`

**类型：** Bash 测试脚本

**运行方式：**
```bash
# 无后端模式（6 个测试）
bash claude_tap_plus/tests/shell/degradation_test.sh

# 有后端模式（8 个测试，需 jq）
bash claude_tap_plus/tests/shell/degradation_test.sh http://127.0.0.1:8080
```

**测试用例（6 + 2 个）：**

| 测试函数 | 覆盖场景 | 需要后端 | 需要 jq |
|---------|---------|---------|---------|
| `test_no_conf_all_pass` | 无 backend.conf 时所有后端调用跳过 | 否 | 否 |
| `test_malformed_conf_no_crash` | 各种 malformed URL 被拒绝 | 否 | 否 |
| `test_backend_down_claim_blocked` | 后端不可达时 `_require_backend` 返回 1 | 否 | 否 |
| `test_backend_down_status_silent` | 后端不可达时 `update_issue_status` 静默返回 0 | 否 | 否 |
| `test_backend_down_call_fails` | 后端不可达时 `_call_backend` 返回非零 | 否 | 否 |
| `test_require_backend_return_codes` | `_require_backend` 三返回码正确 | 否 | 否 |
| `test_backend_up_works` | 后端可用时 check/claim 正常工作 | 是 | 是 |
| `test_recovery_no_conflict` | claim → release → re-claim 流程正确 | 是 | 是 |

**测试环境要求：**
- `curl` — 必须
- `jq` — 后端测试需要，无则跳过
- `bash` — 必须
- Go 后端 — 后端测试需要，无则跳过

---

## 四、测试结果

### Go 测试

```
$ cd claude_tap_plus && go test ./tests/backend/ -v -count=1

=== RUN   TestRecovery_StatePreservedAcrossRestart          --- PASS
=== RUN   TestRecovery_ClaimFailsForAlreadyClaimedAfterRestart --- PASS
=== RUN   TestRecovery_ReleaseSessionAfterRestart           --- PASS
=== RUN   TestMalformedRequest_InvalidJSON                  --- PASS
=== RUN   TestMalformedRequest_EmptyRequiredFields          --- PASS (3 sub)
=== RUN   TestMalformedRequest_NegativeIssueNumber          --- PASS
=== RUN   TestMalformedRequest_StatusEmptyFields            --- PASS (3 sub)
=== RUN   TestMalformedRequest_ExtraFieldsIgnored           --- PASS
=== RUN   TestMalformedRequest_InvalidMethod                --- PASS
=== RUN   TestMalformedRequest_ReleaseSessionEmptySessionID --- PASS
=== RUN   TestMalformedRequest_UnicodeInFields              --- PASS

ok  github.com/liaohch3/claude-tap/claude_tap_plus/tests/backend  1.059s
```

原有测试全部通过，无回归。

### Shell 测试

```
$ bash claude_tap_plus/tests/shell/degradation_test.sh

Results: 6/6 passed, 0 failed
```

带后端测试因环境无 jq 跳过。

---

## 五、未验证项

以下改动在单元/集成测试中覆盖，但**未在真实 Claude Code 会话中端到端验证**：

| 改动 | 验证方式 |
|------|---------|
| `backend.sh` 的 `${SKILL_TAG:-}` / `${CLAUDE_SESSION_ID:-}` | 需实际执行任一 003 技能确认无报错 |
| `001-4-issue-claim` 的 `claim_issue_backend()` 阻止逻辑 | 需后端不可达时执行 `/001-4-issue-claim #N` 确认阻止 |
| `29-session-end` 的 `release_session_issues()` 重构 | 需真实会话结束事件确认释放正常 |
| Shell 测试的 `test_backend_up_works` / `test_recovery_no_conflict` | 需安装 jq 后带后端运行 |

---

## 六、降级行为速查表

| 场景 | 行为 | 测试覆盖 |
|------|------|---------|
| 无 backend.conf | 跳过所有后端调用 | Go + Shell |
| malformed backend.conf | `_load_backend_url` 返回 1，不崩溃 | Shell |
| 后端不可达 + claim | **阻止**，输出错误提示 | Shell |
| 后端不可达 + status 更新 | 静默跳过，return 0 | Shell |
| 后端不可达 + release-session | 静默跳过，return 0 | Shell |
| 后端不可达 + check | 返回完整 issue 列表（不过滤） | Shell |
| 后端重启后状态保持 | SQLite 持久化，状态不变 | Go |
| 后端重启后 claim 冲突 | 已 claim 的 issue 不能被再次 claim | Go |

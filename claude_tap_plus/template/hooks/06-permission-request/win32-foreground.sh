#!/bin/bash
# win32-foreground.sh — Bring terminal to foreground when permission dialog appears
# Trigger: PermissionRequest (hook 06)
# Called from 06-permission-request/base.sh after sourcing hooks/base.sh

# Already have: HOOKS_DIR, HOOK_INPUT, log(), json_get(), OS_TYPE, PYTHON_CMD
FOREGROUND_PS1="$HOOKS_DIR/lib/win32-foreground.ps1"

# Only run on Windows
if [ "$OS_TYPE" != "windows" ]; then
  return 0 2>/dev/null || true
fi

# Log tool info from hook input
tool_name=$(json_get '.tool_name')
tool_input_cmd=""
if [ -n "$PYTHON_CMD" ]; then
  tool_input_cmd=$(printf '%s' "$HOOK_INPUT" | "$PYTHON_CMD" -c "
import json, sys
try:
    d = json.load(sys.stdin)
    ti = d.get('tool_input', {})
    print(ti.get('command', '')[:80] if isinstance(ti, dict) else '')
except: print('')
" 2>/dev/null || echo "")
fi
log "INFO" "[win32-fg] PermissionRequest: tool=$tool_name cmd=${tool_input_cmd:0:80}"

# Bring window to foreground (pass project dir name as hint for multi-window matching)
PROJECT_HINT=$(basename "$CLAUDE_PROJECT_DIR" 2>/dev/null || echo "")
if [ -f "$FOREGROUND_PS1" ]; then
  FOREGROUND_WIN_PATH=$(cygpath -w "$FOREGROUND_PS1" 2>/dev/null || echo "$FOREGROUND_PS1")
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$FOREGROUND_WIN_PATH" -Hint "$PROJECT_HINT" 2>>"$LOG_FILE"
  log "INFO" "[win32-fg] foreground script executed (hint=$PROJECT_HINT)"
else
  log "WARNING" "[win32-fg] win32-foreground.ps1 not found at $FOREGROUND_PS1"
fi

return 0 2>/dev/null || true

#!/bin/bash
# task-complete-notify.sh — Bring terminal to foreground when Claude finishes
# Trigger: Stop event (hook 16)
# Called from 16-stop/base.sh after sourcing hooks/base.sh
# Behavior: Bring window to foreground only (no Toast notification)
# Always exit 0, never block Claude

# Already have: HOOKS_DIR, HOOK_INPUT, log(), json_get(), OS_TYPE, PYTHON_CMD, LOG_FILE
FOREGROUND_PS1="$HOOKS_DIR/lib/win32-foreground.ps1"

# Only run on Windows
if [ "$OS_TYPE" != "windows" ]; then
  return 0 2>/dev/null || true
fi

log "INFO" "[task-notify] Claude task completed, bringing terminal to foreground"

# Bring window to foreground (pass project dir name as hint for multi-window matching)
PROJECT_HINT=$(basename "$CLAUDE_PROJECT_DIR" 2>/dev/null || echo "")
if [ -f "$FOREGROUND_PS1" ]; then
  FOREGROUND_WIN_PATH=$(cygpath -w "$FOREGROUND_PS1" 2>/dev/null || echo "$FOREGROUND_PS1")
  log "INFO" "[task-notify] Calling foreground script (hint=$PROJECT_HINT)"
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$FOREGROUND_WIN_PATH" -Hint "$PROJECT_HINT" 2>>"$LOG_FILE"
  log "INFO" "[task-notify] foreground script done"
else
  log "WARNING" "[task-notify] win32-foreground.ps1 not found at $FOREGROUND_PS1"
fi

return 0 2>/dev/null || true

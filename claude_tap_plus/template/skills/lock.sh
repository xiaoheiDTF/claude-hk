#!/bin/bash
# 文件锁模块 — 使用 mkdir 原子操作（跨平台兼容）
# 用法:
#   source lock.sh
#   lock_acquire "/path/to/lockdir"
#   ... 临界区操作 ...
#   lock_release "/path/to/lockdir"

_LOCK_DIR=""

# 获取锁（阻塞等待，带超时）
# $1 = 锁目录路径
# $2 = 超时秒数（默认 10）
lock_acquire() {
  local lock_dir="$1"
  local timeout="${2:-10}"
  local elapsed=0

  while ! mkdir "$lock_dir" 2>/dev/null; do
    elapsed=$((elapsed + 1))
    if [ "$elapsed" -ge "$timeout" ]; then
      # 超时：检查锁是否是僵尸（超过 60 秒）
      if [ -d "$lock_dir" ]; then
        local lock_age
        lock_age=$(($(date +%s) - $(stat -c %Y "$lock_dir" 2>/dev/null || echo 0)))
        if [ "$lock_age" -gt 60 ]; then
          rm -rf "$lock_dir" 2>/dev/null
          continue
        fi
      fi
      return 1
    fi
    sleep 1
  done

  _LOCK_DIR="$lock_dir"
  return 0
}

# 释放锁
lock_release() {
  [ -n "$_LOCK_DIR" ] && rm -rf "$_LOCK_DIR" 2>/dev/null
  _LOCK_DIR=""
}

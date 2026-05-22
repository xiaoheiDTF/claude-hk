# Changelog

本项目的所有重要变更记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

## [0.5.0] - 2026-05-22

### Added
- 拆分 issue-fix/issue-pr 为单职责 skill（003-5 ~ 003-9）
- 新增 003-6-issue-done：标记开发完成
- 新增 003-8-issue-test：执行 PR Test Plan
- 新增 003-9-issue-review：审核合并或打回
- 优化 Issue/PR/Discussion 模板体系（Markdown + YAML 混合）

### Changed
- 003-5-issue-fix 移除 done 子命令，只保留创建分支
- 003-6-issue-pr 改编号为 003-7-issue-pr，只保留创建 PR

## [0.4.0] - 2026-05-21

### Added
- 细化 issue PR 阶段流程，增加 Test Plan 强制检查机制
- 新增 issue-fix/issue-pr 单职责化拆分方案草稿
- 新增 hooks 和 skill 运行时日志

### Fixed
- Skill 工具边界检查强化（enforce_boundary.sh）

## [0.3.0] - 2026-05-20

### Added
- 003-issues Skill 拆分为 6 个独立 Skill（003-1 ~ 003-6）
- Issue 生命周期标签体系（in-progress, ready-for-pr, rejected）
- 原子领取机制（active.sh + lock.sh 并发安全）

## [0.2.0] - 2026-05-19

### Added
- 004-git-push 和 005-git-commit Skill
- Git 提交规范（中文格式、按类型分组）
- Skill 自动注册（skill-register.sh）

## [0.1.0] - 2026-05-18

### Added
- 项目初始化（init.sh、dirs.conf、ensure_dirs.sh）
- Hooks 管道（29 个生命周期事件）
- 公共基础设施（base.sh、platform.sh、json_get.py）
- 001-testcode-python 和 002-otherdoc Skill
- Skill 注册表（registry.conf）
- 双路日志系统（log.sh）

# Changelog

## [0.2.0] - 2026-06-12

### 新功能

- ✨ **`-s/--setting <profile>`**：直接以指定 profile 启动一个独立的 Claude Code 窗口，
  不污染全局 `~/.claude/settings.json`，可同时开多个窗口分别使用不同 profile。
- ✨ **透传 claude 参数**：通过 `--` 分隔符把任意 claude 参数转发给 claude，
  例如 `claude-switch -s work -- --dangerously-skip-permissions --model opus "提示词"`。
- ✨ **`--claude-bin` flag** 与 **`CLAUDE_BIN` 环境变量**：自定义 claude 可执行文件路径。
- ✨ **无参数运行 `claude-switch`**：交互选择后**直接启动 claude**（不再只是切换 settings.json）。
- ✨ 信号转发（SIGINT/SIGTERM/SIGQUIT/SIGHUP）与子进程退出码透传。
- ✨ 临时 runtime settings 自动清理：进程退出删除；超过 24h 的残留下次启动也会清理。

### 安全

- 🔒 临时 settings 目录权限 `0700`，文件权限 `0600`，避免 API Key 泄漏。
- 🔒 `.gitignore` 默认忽略 `.claude/profiles/.runtime/`。

### 破坏性变更

- 💥 `claude-switch`（无参数）行为从「**仅切换 settings.json**」改为「**交互选择后启动 claude**」。
  老行为迁移到 `claude-switch switch` 子命令；`list` / `init` 子命令保持不变。

### 内部

- ♻️ 抽取 `MarshalCleanProfile` / `FindProfile` 公共函数。
- 📦 新增 `launcher` 包负责拉起 claude 进程。
- 📦 新增 `profile/runtime.go` 负责生成/清理临时 settings。

## [0.1.0]

- 初始版本：交互式 TUI 切换 `~/.claude/settings.json`。

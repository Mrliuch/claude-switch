# Changelog

## [1.3.8] - 2026-06-12

### 新功能

- ✨ **`_args` 字段**：profile JSON 可写默认 args 数组（如 `"_args": ["--dangerously-skip-permissions"]`），
  `-s` 启动时自动追加给 claude，命令行透传参数追加在其后。
- ✨ **双屏交互流程**：无参数运行 `claude-switch` 时，选完 profile 会进入选项勾选面板：
  - YOLO 模式（`--dangerously-skip-permissions`）
  - 续接上次会话（`-c`）
  - 默认勾选状态从 profile `_args` 推断
  - Esc 可返回 profile 选择
- ✨ 选项面板用 `charmbracelet/huh` 实现。

### 内部

- 📦 新增 `ui/options.go`。
- 📦 `Profile` struct 新增 `Args []string` 字段。
- 📦 `cleanProfileData` 同时剔除 `_name` 和 `_args`（不传给 claude）。

## [1.3.7] - 2026-06-12

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

### 行为变更

- ⚠️ `claude-switch`（无参数）行为从「**仅切换 settings.json**」改为「**交互选择后启动 claude**」。
  老行为迁移到 `claude-switch switch` 子命令；`list` / `init` 子命令保持不变。
  如果你的脚本依赖老行为，把 `claude-switch` 改为 `claude-switch switch` 即可。

### 内部

- ♻️ 抽取 `MarshalCleanProfile` / `FindProfile` 公共函数。
- 📦 新增 `launcher` 包负责拉起 claude 进程。
- 📦 新增 `profile/runtime.go` 负责生成/清理临时 settings。

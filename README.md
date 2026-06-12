# claude-switch

Claude Code 配置文件快速切换 / 启动工具，支持官方 OAuth 与多个第三方 API 配置之间切换；
也可以**直接以指定 profile 启动一个独立的 Claude Code 对话窗口**，多个窗口互不干扰。

## 功能

- 🚀 `-s/--setting` 直接启动 claude，**多窗口可同时使用不同 profile，全程不污染全局 `settings.json`**
- 🎯 透传任意 claude 参数（`--dangerously-skip-permissions`、`--model` 等）
- 🖱️ 交互式 TUI 界面，键盘选择配置
- 📦 支持多个 profile（官方 OAuth、第三方 API 等）
- 💾 切换前自动备份当前 `settings.json`（仅 `switch` 子命令）
- 🧹 临时 settings 文件 `0600` 权限，进程退出自动清理

## 安装

```bash
git clone <repo>
cd claude-switch
go build -o claude-switch .
sudo mv claude-switch /usr/local/bin/
```

## 使用

### 直接启动 Claude Code（推荐）

```bash
# 用指定 profile 启动一个 claude 窗口（不影响全局 settings.json）
claude-switch -s provider-a
claude-switch --setting provider-a

# 透传 claude 参数请用 -- 分隔符
claude-switch -s provider-a -- --dangerously-skip-permissions
claude-switch -s provider-a -- --model opus "帮我重构这段代码"
claude-switch -s provider-a -- -c                             # claude 的 -c (continue)

# 不指定 profile：双屏交互（选 profile → 勾选启动选项 → 启动）
claude-switch

# 自定义 claude 可执行文件路径
claude-switch -s provider-a --claude-bin /opt/claude/bin/claude
CLAUDE_BIN=/opt/claude/bin/claude claude-switch -s provider-a
```

### 双屏交互流程（无参数运行）

```
第一屏：选 profile                第二屏：勾选启动选项
┌──────────────────────┐          ┌─────────────────────────────────────┐
│ > 闻歌-云计算-刘晨 ●  │          │ 启动选项 - 闻歌-云计算-刘晨          │
│   Luken-NewAPI       │   Enter  │                                      │
│   小马API            │  ──────> │ [✓] YOLO 模式（--dangerously-...）  │
│   官方 OAuth         │          │ [ ] 续接上次会话（-c）              │
└──────────────────────┘          │                                      │
                                   │ 空格切换  Enter 启动  Esc 返回      │
                                   └─────────────────────────────────────┘
```

- 第二屏的默认勾选状态从 profile 的 `_args` 字段推断（详见下文）
- Esc 可从第二屏返回第一屏重选 profile

### 在 profile 里写默认参数（_args）

如果某个 profile 每次都要带 YOLO，可以在 profile JSON 里加 `_args` 字段：

```json
{
  "_name": "闻歌-云计算-刘晨",
  "_args": ["--dangerously-skip-permissions"],
  "env": { "ANTHROPIC_BASE_URL": "...", "ANTHROPIC_API_KEY": "..." }
}
```

- `claude-switch -s 闻歌` 会自动在启动 claude 时追加 `--dangerously-skip-permissions`
- 命令行 `--` 之后的参数会**追加在 `_args` 之后**：
  ```bash
  claude-switch -s 闻歌 -- --model opus
  # 实际执行：claude --settings xxx --dangerously-skip-permissions --model opus
  ```
- 交互模式下 `_args` 会作为「默认勾选状态」反映到选项面板，可临时取消

可以同时打开多个终端窗口，每个用不同 profile，互不冲突：

```bash
# 终端 1
claude-switch -s work

# 终端 2
claude-switch -s personal -- --dangerously-skip-permissions

# 终端 3
claude-switch -s provider-a
```

### 切换全局 settings.json（v0.1 行为）

```bash
# 交互选择并写入 ~/.claude/settings.json + 更新 .current
claude-switch switch

# 列出所有可用配置
claude-switch list

# 重新初始化默认配置（将当前 settings.json 存为默认 profile）
claude-switch init
```

### ⚠️ v1.3.7 行为变更

| 老命令 | v1.3.7 等效命令 |
|---|---|
| `claude-switch`（仅切换 settings.json） | `claude-switch switch` |
| `claude-switch`（新默认行为） | 交互选择后**直接启动 claude**，不改全局 settings.json |

如果你的脚本依赖老行为，把 `claude-switch` 改为 `claude-switch switch` 即可。

## 配置文件管理

所有 profile 存放在 `~/.claude/profiles/` 目录，每个文件是完整的 `settings.json` 内容，额外增加 `_name` 字段作为显示名称：

```json
{
  "_name": "第三方 Provider A",
  "env": {
    "ANTHROPIC_BASE_URL": "https://api.provider-a.com",
    "ANTHROPIC_API_KEY": "sk-xxxx"
  }
}
```

> **⚠️ 安全提示**：profile 文件中含明文 API Key，请妥善保管：
> - 不要提交到公开仓库
> - 文件权限建议 `chmod 600`
> - 临时 runtime settings（`~/.claude/profiles/.runtime/`）已默认 `0600`，进程退出自动删除

### 目录结构

```
~/.claude/
├── settings.json          # 当前生效配置（仅 `switch` 子命令会改）
├── settings.json.bak      # 上次切换前的备份
└── profiles/
    ├── .current           # 当前激活的 profile 文件名（仅 switch 子命令更新）
    ├── .runtime/          # -s 启动时生成的临时 settings（0700 目录、0600 文件）
    │   └── runtime-*.json # 进程退出自动清理；超过 24h 的残留下次启动也会清理
    ├── anthropic-oauth.json
    ├── provider-a.json
    └── provider-b.json
```

## 工作原理

`claude-switch -s xxx` 的执行流程：

1. 在 `~/.claude/profiles/.runtime/` 生成一个去掉 `_name` 字段的临时 settings JSON（`0600` 权限）
2. 调用 `claude --settings <临时文件> [透传参数...]`
3. 子进程的信号（SIGINT/SIGTERM/SIGQUIT/SIGHUP）会被父进程透传
4. 子进程退出时，临时 settings 文件自动删除；父进程透传子进程退出码

因此每个窗口都是完全独立的 claude 实例，全局 `~/.claude/settings.json` **不会被改动**。

## 键盘操作（交互模式）

| 按键 | 功能 |
|------|------|
| ↑ / k | 向上移动 |
| ↓ / j | 向下移动 |
| Enter | 确认 |
| q / Esc / Ctrl+C | 退出 |

## 常见问题

**Q: 透传参数为什么要加 `--`？**
A: 为了让 cobra 不把 `--dangerously-skip-permissions` 当成 claude-switch 自己的 flag。`--` 是 POSIX 标准做法（git/docker/kubectl 都这么干）。

**Q: 没装 claude 怎么办？**
A: 工具会给出官方安装文档链接，也支持 `--claude-bin` 或 `CLAUDE_BIN` 环境变量指定自定义路径。

**Q: 同时开 5 个窗口会创建 5 份临时 settings 吗？**
A: 是的，每份带 PID + 随机后缀互不冲突，进程退出自动删除。即使崩溃没清理，下次启动会清理超过 24h 的残留。

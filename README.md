# claude-switch

Claude Code 配置文件快速切换工具，支持官方 OAuth 与多个第三方 API 配置之间的交互式切换。

## 功能

- 交互式 TUI 界面，键盘选择配置
- 支持多个 profile（官方 OAuth、第三方 API 等）
- 切换前自动备份当前 `settings.json`
- 首次运行自动初始化默认配置

## 安装

```bash
git clone <repo>
cd claude-switch
go build -o claude-switch .
sudo mv claude-switch /usr/local/bin/
```

## 使用

```bash
# 启动交互式选择界面
claude-switch

# 列出所有可用配置
claude-switch list

# 重新初始化默认配置（将当前 settings.json 存为默认 profile）
claude-switch init
```

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

> **注意**：profile 文件中含有明文 API Key，请妥善保管，不要提交到公开仓库。

### 目录结构

```
~/.claude/
├── settings.json          # 当前生效配置（由工具自动管理）
├── settings.json.bak      # 上次切换前的备份
└── profiles/
    ├── .current           # 当前激活的 profile 文件名
    ├── anthropic-oauth.json
    ├── provider-a.json
    └── provider-b.json
```

## 键盘操作

| 按键 | 功能 |
|------|------|
| ↑ / k | 向上移动 |
| ↓ / j | 向下移动 |
| Enter | 确认切换 |
| q / Esc | 退出 |

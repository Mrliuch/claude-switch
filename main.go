package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/liuchen/claude-switch/launcher"
	"github.com/liuchen/claude-switch/profile"
	"github.com/liuchen/claude-switch/ui"
	"github.com/spf13/cobra"
)

var (
	flagSetting   string
	flagClaudeBin string
)

var rootCmd = &cobra.Command{
	Use:   "claude-switch [-- <claude args>...]",
	Short: "Claude Code 配置文件快速切换 / 启动工具",
	Long: `通过交互式界面快速切换 Claude Code 的 settings.json 配置，
也可以直接以指定 profile 启动一个独立的 Claude Code 对话窗口
（互不干扰全局 settings.json，可同时开多个）。

透传 claude 参数请使用 -- 分隔符：
  claude-switch -s work -- --dangerously-skip-permissions
  claude-switch -s work -- --model opus "提示词"`,
	RunE:               run,
	SilenceUsage:       true,
	SilenceErrors:      true,
	DisableFlagParsing: false,
}

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "列出所有可用配置",
	RunE:         listProfiles,
	SilenceUsage: true,
}

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "将当前 settings.json 初始化为默认 profile",
	RunE:         initProfile,
	SilenceUsage: true,
}

var switchCmd = &cobra.Command{
	Use:          "switch",
	Short:        "交互选择并将 profile 写入全局 settings.json（不启动 claude）",
	Long:         "原 v0.1 默认行为：交互选择 profile，写入 ~/.claude/settings.json 并更新 .current 记录。不会启动 claude。",
	RunE:         runSwitch,
	SilenceUsage: true,
}

// run 是默认行为：
//   - 带 -s 参数：直接以指定 profile 启动 claude
//   - 不带 -s：交互选择后启动 claude
//
// 不会修改全局 settings.json，也不会更新 .current。
func run(cmd *cobra.Command, args []string) error {
	if profile.IsProfilesDirEmpty() {
		return bootstrapDefault()
	}

	var chosen *profile.Profile

	if flagSetting != "" {
		p, err := profile.FindProfile(flagSetting)
		if err != nil {
			return err
		}
		chosen = p
	} else {
		profiles, err := profile.LoadProfiles()
		if err != nil {
			return err
		}
		current := profile.GetCurrentProfile()
		picked, err := ui.Run(profiles, current)
		if err != nil {
			return err
		}
		if picked == nil {
			fmt.Println("已取消")
			return nil
		}
		chosen = picked
	}

	return launchWithProfile(*chosen, args)
}

// runSwitch 是 v0.1 的默认行为，迁移到 switch 子命令。
func runSwitch(cmd *cobra.Command, args []string) error {
	if profile.IsProfilesDirEmpty() {
		return bootstrapDefault()
	}

	profiles, err := profile.LoadProfiles()
	if err != nil {
		return err
	}

	current := profile.GetCurrentProfile()
	chosen, err := ui.Run(profiles, current)
	if err != nil {
		return err
	}

	if chosen == nil {
		fmt.Println("已取消")
		return nil
	}

	if chosen.Filename == current {
		fmt.Printf("已是当前配置: %s\n", chosen.Name)
		return nil
	}

	if err := profile.SwitchProfile(*chosen); err != nil {
		return err
	}

	fmt.Printf("✓ 切换成功: %s\n", chosen.Name)
	fmt.Printf("  已备份原配置到 settings.json.bak\n")
	return nil
}

// launchWithProfile 渲染临时 settings 并拉起 claude，返回时透传退出码。
func launchWithProfile(p profile.Profile, extraArgs []string) error {
	claudeBin, err := launcher.ResolveClaudeBin(flagClaudeBin)
	if err != nil {
		return err
	}

	settingsPath, cleanup, err := profile.RenderRuntimeSettings(p)
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Fprintf(os.Stderr, "▶ 使用 profile: %s [%s]\n", p.Name, p.Filename)

	code, err := launcher.Launch(claudeBin, settingsPath, extraArgs)
	if err != nil {
		return err
	}
	if code != 0 {
		// 通过 ExitError 让 cobra 不打印错误信息，main 里用退出码退出。
		return &exitCodeError{code: code}
	}
	return nil
}

func bootstrapDefault() error {
	fmt.Println("首次运行，正在初始化默认配置...")
	if err := profile.InitDefaultProfile(); err != nil {
		return err
	}
	fmt.Printf("✓ 已将当前 settings.json 保存为「官方 OAuth（默认配置）」\n")
	fmt.Printf("  配置目录: %s\n", profile.ProfilesPath())
	fmt.Println("  可向该目录添加更多 .json 配置文件，然后重新运行本工具")
	return nil
}

func listProfiles(cmd *cobra.Command, args []string) error {
	if profile.IsProfilesDirEmpty() {
		fmt.Println("暂无配置文件，请先运行 claude-switch 进行初始化")
		return nil
	}

	profiles, err := profile.LoadProfiles()
	if err != nil {
		return err
	}

	current := profile.GetCurrentProfile()
	fmt.Printf("配置目录: %s\n\n", profile.ProfilesPath())
	for _, p := range profiles {
		mark := "  "
		if p.Filename == current {
			mark = "▶ "
		}
		fmt.Printf("%s%s  [%s]\n", mark, p.Name, p.Filename)
	}
	return nil
}

func initProfile(cmd *cobra.Command, args []string) error {
	if err := profile.InitDefaultProfile(); err != nil {
		return err
	}
	fmt.Printf("✓ 已将当前 settings.json 保存为「官方 OAuth（默认配置）」\n")
	fmt.Printf("  配置目录: %s\n", profile.ProfilesPath())
	return nil
}

// exitCodeError 仅用于把子进程退出码传到 main，不打印错误信息。
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exit code %d", e.code) }

func main() {
	// -s 之后的位置参数（如 "--" 之后的内容）保持原顺序，不当 flag 解析。
	rootCmd.Flags().SetInterspersed(false)

	rootCmd.PersistentFlags().StringVarP(&flagSetting, "setting", "s", "",
		"指定 profile 文件名（不含 .json 后缀），直接启动 claude 而不进入交互界面")
	rootCmd.PersistentFlags().StringVar(&flagClaudeBin, "claude-bin", "",
		"指定 claude 可执行文件路径（默认从 PATH 查找，亦可通过 CLAUDE_BIN 环境变量设置）")

	rootCmd.AddCommand(listCmd, initCmd, switchCmd)

	if err := rootCmd.Execute(); err != nil {
		var ec *exitCodeError
		if errors.As(err, &ec) {
			os.Exit(ec.code)
		}
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/liuchen/claude-switch/profile"
	"github.com/liuchen/claude-switch/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude-switch",
	Short: "Claude Code 配置文件快速切换工具",
	Long:  "通过交互式界面快速切换 Claude Code 的 settings.json 配置，支持官方 OAuth 与多个第三方 API 配置之间的切换。",
	RunE:  run,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有可用配置",
	RunE:  listProfiles,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "将当前 settings.json 初始化为默认 profile",
	RunE:  initProfile,
}

func run(cmd *cobra.Command, args []string) error {
	if profile.IsProfilesDirEmpty() {
		fmt.Println("首次运行，正在初始化默认配置...")
		if err := profile.InitDefaultProfile(); err != nil {
			return err
		}
		fmt.Printf("✓ 已将当前 settings.json 保存为「官方 OAuth（默认配置）」\n")
		fmt.Printf("  配置目录: %s\n", profile.ProfilesPath())
		fmt.Println("  可向该目录添加更多 .json 配置文件，然后重新运行本工具")
		return nil
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

func main() {
	rootCmd.AddCommand(listCmd, initCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

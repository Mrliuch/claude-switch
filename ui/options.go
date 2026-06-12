package ui

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/liuchen/claude-switch/profile"
)

// 选项面板内置的开关定义。每个开关对应一个或多个 claude 命令行参数。
type optionDef struct {
	id     string   // 内部标识
	label  string   // 显示给用户的文字
	args   []string // 勾选后追加给 claude 的参数
	persist bool    // 默认勾选状态是否要从 profile.Args 读出（不持久的开关每次默认不勾选）
}

var optionDefs = []optionDef{
	{
		id:      "yolo",
		label:   "YOLO 模式（--dangerously-skip-permissions：跳过所有权限确认）",
		args:    []string{"--dangerously-skip-permissions"},
		persist: true,
	},
	{
		id:      "continue",
		label:   "续接上次会话（-c / --continue）",
		args:    []string{"-c"},
		persist: false, // 续接是临时操作，每次默认不勾选
	},
}

// ErrOptionsAborted 表示用户在选项面板按下 Esc / Ctrl+C 取消。
var ErrOptionsAborted = errors.New("用户取消选项面板")

// RunOptions 显示一个 MultiSelect 面板让用户调整启动参数。
//
// 默认勾选状态：
//   - persist=true 的开关：profile.Args 里出现就默认勾选（如 _args 里写了 YOLO）
//   - persist=false 的开关：始终默认不勾选
//
// 返回最终要追加给 claude 的参数（合并选中开关的 args + profile.Args 中
// 不属于任何已知开关的「自定义参数」，保持原顺序）。
func RunOptions(p profile.Profile) ([]string, error) {
	defaults := defaultSelections(p.Args)
	custom := customArgs(p.Args)

	selected := make([]string, 0, len(defaults))
	selected = append(selected, defaults...)

	options := make([]huh.Option[string], 0, len(optionDefs))
	for _, def := range optionDefs {
		opt := huh.NewOption(def.label, def.id)
		if contains(defaults, def.id) {
			opt = opt.Selected(true)
		}
		options = append(options, opt)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(fmt.Sprintf("启动选项 - %s", p.Name)).
				Description("空格切换  Enter 启动  Esc/Ctrl+C 取消").
				Options(options...).
				Filterable(false).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, ErrOptionsAborted
		}
		return nil, err
	}

	return composeArgs(selected, custom), nil
}

// defaultSelections 根据 profile.Args 计算默认勾选的开关 id。
func defaultSelections(profileArgs []string) []string {
	if len(profileArgs) == 0 {
		return nil
	}
	matched := make([]string, 0, len(optionDefs))
	for _, def := range optionDefs {
		if !def.persist {
			continue
		}
		if argsContainsAny(profileArgs, def.args) {
			matched = append(matched, def.id)
		}
	}
	return matched
}

// customArgs 返回 profile.Args 中不属于任何已知开关的自定义参数。
// 这部分参数无论用户怎么勾选，都会保留追加给 claude。
func customArgs(profileArgs []string) []string {
	if len(profileArgs) == 0 {
		return nil
	}
	known := make(map[string]struct{})
	for _, def := range optionDefs {
		for _, a := range def.args {
			known[a] = struct{}{}
		}
	}
	custom := make([]string, 0, len(profileArgs))
	for _, a := range profileArgs {
		if _, ok := known[a]; ok {
			continue
		}
		custom = append(custom, a)
	}
	return custom
}

// composeArgs 把选中的开关 id 翻译成 args，再附加 profile 的自定义 args。
func composeArgs(selectedIDs, custom []string) []string {
	out := make([]string, 0, len(selectedIDs)+len(custom))
	for _, id := range selectedIDs {
		for _, def := range optionDefs {
			if def.id == id {
				out = append(out, def.args...)
				break
			}
		}
	}
	out = append(out, custom...)
	return out
}

func argsContainsAny(haystack, needles []string) bool {
	for _, n := range needles {
		if contains(haystack, n) {
			return true
		}
	}
	return false
}

func contains(s []string, target string) bool {
	for _, item := range s {
		if item == target {
			return true
		}
	}
	return false
}

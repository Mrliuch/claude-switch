package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/liuchen/claude-switch/profile"
)

var (
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Bold(true).
			Foreground(lipgloss.Color("205"))

	activeStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("42")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color("250"))

	dimStyle = lipgloss.NewStyle().
			PaddingLeft(6).
			Foreground(lipgloss.Color("240"))
)

type profileItem struct {
	p       profile.Profile
	current bool
}

func (i profileItem) Title() string {
	if i.current {
		return fmt.Sprintf("● %s  [%s]", i.p.Name, i.p.Filename)
	}
	return fmt.Sprintf("○ %s  [%s]", i.p.Name, i.p.Filename)
}

func (i profileItem) Description() string {
	if i.current {
		return "  当前激活"
	}
	return ""
}

func (i profileItem) FilterValue() string { return i.p.Name + " " + i.p.Filename }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(profileItem)
	if !ok {
		return
	}

	str := i.Title()
	if index == m.Index() {
		str = "> " + str
		if i.current {
			fmt.Fprint(w, activeStyle.Render(str))
		} else {
			fmt.Fprint(w, lipgloss.NewStyle().PaddingLeft(2).Bold(true).Foreground(lipgloss.Color("170")).Render(str))
		}
	} else {
		if i.current {
			fmt.Fprint(w, activeStyle.PaddingLeft(4).Render(str))
		} else {
			fmt.Fprint(w, normalStyle.Render(str))
		}
	}
}

type model struct {
	list     list.Model
	chosen   *profile.Profile
	quitting bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if selected, ok := m.list.SelectedItem().(profileItem); ok {
				m.chosen = &selected.p
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View() + "\n\n  ↑↓/jk 移动   Enter 确认   q 退出\n"
}

func Run(profiles []profile.Profile, current string) (*profile.Profile, error) {
	items := make([]list.Item, len(profiles))
	defaultIndex := 0
	for i, p := range profiles {
		isCurrent := p.Filename == current
		items[i] = profileItem{p: p, current: isCurrent}
		if isCurrent {
			defaultIndex = i
		}
	}

	height := len(profiles) + 4
	if height < 8 {
		height = 8
	}

	l := list.New(items, itemDelegate{}, 70, height)
	l.Title = "Claude Config Switcher"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Select(defaultIndex)

	m := model{list: l}
	prog := tea.NewProgram(m)
	result, err := prog.Run()
	if err != nil {
		return nil, err
	}

	final := result.(model)
	if final.quitting || final.chosen == nil {
		return nil, nil
	}
	return final.chosen, nil
}

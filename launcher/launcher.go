package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const (
	// claudeBinEnv 允许通过环境变量指定 claude 可执行文件路径。
	claudeBinEnv = "CLAUDE_BIN"
	// 官方安装文档链接，给找不到 claude 时的提示用。
	installDocURL = "https://docs.claude.com/en/docs/claude-code/quickstart"
)

// ErrClaudeNotFound 表示无法在 PATH 或环境变量中定位 claude 可执行文件。
var ErrClaudeNotFound = errors.New("未找到 claude 可执行文件")

// ResolveClaudeBin 按优先级解析 claude 可执行文件路径：
//  1. override 参数（来自 --claude-bin flag）
//  2. CLAUDE_BIN 环境变量
//  3. PATH 中的 claude
func ResolveClaudeBin(override string) (string, error) {
	if override != "" {
		return resolveExplicit(override, "--claude-bin")
	}
	if env := os.Getenv(claudeBinEnv); env != "" {
		return resolveExplicit(env, claudeBinEnv)
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("%w；请先安装 Claude Code CLI（参考 %s），或通过 --claude-bin / %s 环境变量显式指定路径",
			ErrClaudeNotFound, installDocURL, claudeBinEnv)
	}
	return path, nil
}

func resolveExplicit(path, source string) (string, error) {
	resolved, err := exec.LookPath(path)
	if err != nil {
		return "", fmt.Errorf("通过 %s 指定的 claude 路径 %q 不可执行: %w", source, path, err)
	}
	return resolved, nil
}

// Launch 以 settingsPath 作为 --settings 参数拉起 claude，并把 extraArgs 透传给 claude。
//
// stdin/stdout/stderr 直接复用当前进程，信号会转发给子进程，
// 返回 claude 的退出码（被信号终止时返回 128 + signal）。
func Launch(claudeBin, settingsPath string, extraArgs []string) (int, error) {
	if claudeBin == "" {
		return 1, errors.New("claude 可执行文件路径为空")
	}
	if settingsPath == "" {
		return 1, errors.New("settings 路径为空")
	}

	args := append([]string{"--settings", settingsPath}, extraArgs...)

	cmd := exec.Command(claudeBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("启动 claude 失败: %w", err)
	}

	// 转发信号到子进程：子进程是 TUI，需要它自行处理 SIGINT 等。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	doneCh := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-doneCh:
				return
			}
		}
	}()

	err := cmd.Wait()
	close(doneCh)
	signal.Stop(sigCh)

	return exitCodeFromError(err), nil
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return 128 + int(status.Signal())
			}
			return status.ExitStatus()
		}
		return exitErr.ExitCode()
	}
	return 1
}

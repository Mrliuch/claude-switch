package profile

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	runtimeDirName    = ".runtime"
	runtimeFilePrefix = "runtime-"
	runtimeFileSuffix = ".json"
	// 残留文件保留时间：超过此时长的会在下次启动时清理。
	runtimeStaleAfter = 24 * time.Hour
)

// runtimeDir 返回临时 settings 文件所在目录。
func runtimeDir() string {
	return filepath.Join(ProfilesPath(), runtimeDirName)
}

// RenderRuntimeSettings 把 profile 渲染成一个独立的临时 settings.json，
// 返回文件路径以及 cleanup 函数（调用方应 defer cleanup）。
//
// 文件位于 ~/.claude/profiles/.runtime/，目录权限 0700，文件权限 0600，
// 内容是剔除 _name 字段后的纯净 settings JSON。
//
// 该函数也会顺手清理超过 runtimeStaleAfter 的历史残留，避免崩溃留下垃圾。
func RenderRuntimeSettings(p Profile) (string, func(), error) {
	dir := runtimeDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, fmt.Errorf("创建 runtime 目录失败: %w", err)
	}

	// 顺手清理过期残留（错误忽略：清理失败不应阻塞主流程）。
	_ = sweepStaleRuntimeFiles(dir, runtimeStaleAfter)

	out, err := MarshalCleanProfile(p)
	if err != nil {
		return "", nil, err
	}

	suffix, err := randomSuffix()
	if err != nil {
		return "", nil, fmt.Errorf("生成随机后缀失败: %w", err)
	}

	filename := fmt.Sprintf("%s%s-%d-%s%s",
		runtimeFilePrefix,
		sanitizeName(p.Filename),
		os.Getpid(),
		suffix,
		runtimeFileSuffix,
	)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, out, 0600); err != nil {
		return "", nil, fmt.Errorf("写入 runtime settings 失败: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(path)
	}
	return path, cleanup, nil
}

func randomSuffix() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// sanitizeName 防止 profile 文件名里含有路径分隔符等危险字符。
func sanitizeName(name string) string {
	if name == "" {
		return "profile"
	}
	replacer := strings.NewReplacer(
		string(os.PathSeparator), "_",
		"/", "_",
		"\\", "_",
		"..", "_",
	)
	cleaned := replacer.Replace(name)
	if cleaned == "" {
		return "profile"
	}
	return cleaned
}

// sweepStaleRuntimeFiles 清理 dir 下超过 staleAfter 的 runtime-*.json 文件。
func sweepStaleRuntimeFiles(dir string, staleAfter time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-staleAfter)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, runtimeFilePrefix) || !strings.HasSuffix(name, runtimeFileSuffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}

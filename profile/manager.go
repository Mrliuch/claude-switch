package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	nameKey         = "_name"
	argsKey         = "_args"
	profilesDirName = ".claude/profiles"
	settingsName    = ".claude/settings.json"
	backupName      = ".claude/settings.json.bak"
	currentFileName = ".current"
	defaultFilename = "anthropic-oauth"
	defaultName     = "官方 OAuth（默认配置）"
)

type Profile struct {
	Name     string
	Filename string
	// Args 是 profile 在启动 claude 时默认追加的参数（来自 _args 字段）。
	// 例：["--dangerously-skip-permissions"] 会让 -s 启动时自动带 YOLO。
	Args []string
	Data map[string]interface{}
}

func ProfilesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, profilesDirName)
}

func SettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, settingsName)
}

func backupPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, backupName)
}

func LoadProfiles() ([]Profile, error) {
	dir := ProfilesPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取 profiles 目录失败: %w", err)
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		filename := strings.TrimSuffix(entry.Name(), ".json")
		displayName := filename
		if n, ok := raw[nameKey].(string); ok && n != "" {
			displayName = n
		}

		profiles = append(profiles, Profile{
			Name:     displayName,
			Filename: filename,
			Args:     parseArgs(raw[argsKey]),
			Data:     raw,
		})
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("profiles 目录中没有有效的 .json 配置文件")
	}
	return profiles, nil
}

func GetCurrentProfile() string {
	data, err := os.ReadFile(filepath.Join(ProfilesPath(), currentFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// cleanProfileData 返回剔除元数据字段（_name / _args）后的新 map（不修改原数据）。
// 这些字段是 claude-switch 自用的元数据，不应该传给 claude。
func cleanProfileData(data map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(data))
	for k, v := range data {
		if k == nameKey || k == argsKey {
			continue
		}
		clean[k] = v
	}
	return clean
}

// parseArgs 把 JSON 里的 _args 字段（可能是 nil / 任意类型 / 字符串数组）
// 转成 []string，类型不对时返回 nil（即没有默认参数）。
func parseArgs(v interface{}) []string {
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	args := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			args = append(args, s)
		}
	}
	if len(args) == 0 {
		return nil
	}
	return args
}

// MarshalCleanProfile 把 profile 数据剔除 _name 后序列化为缩进 JSON。
func MarshalCleanProfile(p Profile) ([]byte, error) {
	out, err := json.MarshalIndent(cleanProfileData(p.Data), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化失败: %w", err)
	}
	return out, nil
}

func SwitchProfile(p Profile) error {
	settingsPath := SettingsPath()

	// 备份当前 settings.json
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := os.WriteFile(backupPath(), data, 0644); err != nil {
			return fmt.Errorf("备份失败: %w", err)
		}
	}

	out, err := MarshalCleanProfile(p)
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("写入 settings.json 失败: %w", err)
	}

	return os.WriteFile(
		filepath.Join(ProfilesPath(), currentFileName),
		[]byte(p.Filename),
		0644,
	)
}

// FindProfile 根据文件名（不含 .json 后缀）查找 profile。
func FindProfile(filename string) (*Profile, error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		if profiles[i].Filename == filename {
			return &profiles[i], nil
		}
	}
	return nil, fmt.Errorf("未找到名为 %q 的 profile，请用 `claude-switch list` 查看可用配置", filename)
}

func IsProfilesDirEmpty() bool {
	entries, err := os.ReadDir(ProfilesPath())
	if os.IsNotExist(err) {
		return true
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			return false
		}
	}
	return true
}

func InitDefaultProfile() error {
	if err := os.MkdirAll(ProfilesPath(), 0755); err != nil {
		return fmt.Errorf("创建 profiles 目录失败: %w", err)
	}

	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		return fmt.Errorf("读取 settings.json 失败: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析 settings.json 失败: %w", err)
	}

	raw[nameKey] = defaultName

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化默认 profile 失败: %w", err)
	}

	profilePath := filepath.Join(ProfilesPath(), defaultFilename+".json")
	if err := os.WriteFile(profilePath, out, 0644); err != nil {
		return fmt.Errorf("写入默认 profile 失败: %w", err)
	}

	return os.WriteFile(
		filepath.Join(ProfilesPath(), currentFileName),
		[]byte(defaultFilename),
		0644,
	)
}

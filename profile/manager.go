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
	Data     map[string]interface{}
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

func SwitchProfile(p Profile) error {
	settingsPath := SettingsPath()

	// 备份当前 settings.json
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := os.WriteFile(backupPath(), data, 0644); err != nil {
			return fmt.Errorf("备份失败: %w", err)
		}
	}

	// 剔除 _name 字段后写入 settings.json
	clean := make(map[string]interface{}, len(p.Data))
	for k, v := range p.Data {
		if k != nameKey {
			clean[k] = v
		}
	}

	out, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
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
		return fmt.Errorf("序列化失败: %w", err)
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

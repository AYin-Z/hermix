package locales

import (
	"bbs-go/internal/pkg/config"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var viperInstances = make(map[string]*viper.Viper)

func getLocaleDir() string {
	workDir, err := os.Executable()
	if err != nil {
		return "./locales"
	}

	localeDir := filepath.Join(filepath.Dir(workDir), "locales")
	if _, err := os.Stat(localeDir); err == nil {
		return localeDir
	}

	return "./locales"
}

func Init() error {
	files, err := filepath.Glob(filepath.Join(getLocaleDir(), "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to find locale files: %w", err)
	}

	for _, file := range files {
		v := viper.New()
		v.SetConfigFile(file)
		v.SetConfigType("yaml")
		v.WatchConfig()

		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read locale file %s: %w", file, err)
		}

		locale := getLocaleByFile(file)
		viperInstances[locale] = v
	}

	return nil
}

func getLocaleByFile(file string) string {
	if strings.HasSuffix(file, ".yml") {
		return strings.TrimSuffix(filepath.Base(file), ".yml")
	} else if strings.HasSuffix(file, ".yaml") {
		return strings.TrimSuffix(filepath.Base(file), ".yaml")
	}
	return ""
}

func Get(key string) string {
	v, ok := viperInstances[string(config.Instance.Language)]
	if !ok {
		slog.Error("locale not found", "locale", config.Instance.Language)
		return key
	}

	if !v.IsSet(key) {
		slog.Warn("translation key not found", "key", key, "locale", config.Instance.Language)
		return key
	}

	return v.GetString(key)
}

func Getf(key string, args ...any) string {
	format := Get(key)
	return fmt.Sprintf(format, args...)
}

// localizedError 是身份稳定、文案惰性本地化的错误。
// 指针相等 / errors.Is 照常可用；Error() 在调用时才按当前语言取文案，
// 因此可安全地作为包级哨兵变量初始化（此时 locales 尚未 Init 也没关系）。
type localizedError struct{ key string }

func (e *localizedError) Error() string { return Get(e.key) }

// NewError 返回一个文案惰性本地化的哨兵错误。
func NewError(key string) error { return &localizedError{key: key} }

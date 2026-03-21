package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DataDir   string `mapstructure:"data_dir"`
	ClaudeBin string `mapstructure:"claude_bin"`
}

func Load(cfgFile string) (*Config, error) {
	// defaults
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, ".muxc")

	viper.SetDefault("data_dir", defaultDataDir)
	viper.SetDefault("claude_bin", "") // auto-detect from PATH

	viper.SetEnvPrefix("MUXC")
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(defaultDataDir)
	}

	viper.ReadInConfig() // ignore error — config file is optional

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) SessionsDir() string {
	return filepath.Join(c.DataDir, "sessions")
}

// GetClaudeBin returns the configured claude binary or finds it in PATH.
func (c *Config) GetClaudeBin() (string, error) {
	if c.ClaudeBin != "" {
		return c.ClaudeBin, nil
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH; set claude_bin in config or MUXC_CLAUDE_BIN env var")
	}
	return path, nil
}

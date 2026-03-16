package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"
)

// Config holds the contents of the user's ~/.sleeper/config.yaml.
type Config struct {
	Leagues map[string]string `mapstructure:"leagues"`
}

// Load reads the config from cfgFile, or from ~/.sleeper/config.yaml if empty.
func Load(cfgFile string) (Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("could not determine home directory: %w", err)
		}
		viper.AddConfigPath(filepath.Join(home, ".sleeper"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("could not read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("could not parse config: %w", err)
	}

	return cfg, nil
}

// Years returns the configured season years in descending order.
func (c Config) Years() []string {
	years := make([]string, 0, len(c.Leagues))
	for y := range c.Leagues {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(years)))
	return years
}

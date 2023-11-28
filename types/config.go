package types

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fioncat/grfs/osutils"
	"gopkg.in/yaml.v3"
)

const (
	configMinimalDuration = time.Millisecond * 100
	configMaximalDuration = time.Minute * 10

	configDefaultOpenBoltTimeout = time.Second * 3
	configDefaultFsTimeout       = time.Second * 10
)

type Config struct {
	BaseDir string `yaml:"-"`
	Path    string `yaml:"-"`

	OpenBoltTimeout time.Duration `yaml:"openBoltTimeout"`

	Fs *FilesystemConfig `yaml:"fs"`

	Auths Auths `yaml:"auths"`
}

type Auths map[string]string

type FilesystemConfig struct {
	AllowOthers  bool          `yaml:"allowOthers"`
	EntryTimeout time.Duration `yaml:"entryTimeout"`

	Debug bool `yaml:"debug"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := getConfigPath(homeDir)

	baseDir := os.Getenv("GRFS_BASE_PATH")
	if baseDir == "" {
		baseDir = filepath.Join(homeDir, ".local", "share", "grfs")
	}
	err = osutils.EnsureDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("ensure basedir: %w", err)
	}

	if path == "" {
		return newDefaultConfig(path, baseDir), nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newDefaultConfig(path, baseDir), nil
		}

		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("decode config yaml file: %w", err)
	}

	err = cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	cfg.BaseDir = baseDir
	cfg.Path = path

	return &cfg, nil
}

func getConfigPath(homeDir string) string {
	path := os.Getenv("GRFS_CONFIG_PATH")
	if path != "" {
		return path
	}
	dir := filepath.Join(homeDir, ".config", "grfs")
	ents, err := os.ReadDir(dir)
	if err == nil {
		for _, ent := range ents {
			switch ent.Name() {
			case "config.yaml", "config.yml":
				return filepath.Join(dir, ent.Name())
			}
		}
	}
	return ""
}

func newDefaultConfig(path, baseDir string) *Config {
	c := &Config{
		BaseDir: baseDir,
		Path:    path,

		OpenBoltTimeout: configDefaultOpenBoltTimeout,

		Auths: make(Auths),
	}
	c.Fs = c.newDefaultFilesystem()

	return c
}

func (c *Config) validate() error {
	if c.OpenBoltTimeout > 0 {
		err := c.validateDuration(c.OpenBoltTimeout)
		if err != nil {
			return fmt.Errorf("invalid openBoltTimeout: %w", err)
		}
	} else {
		c.OpenBoltTimeout = configDefaultOpenBoltTimeout
	}

	if c.Auths != nil {
		for key, token := range c.Auths {
			c.Auths[key] = os.ExpandEnv(token)
		}
	}
	if c.Fs == nil {
		c.Fs = c.newDefaultFilesystem()
	}

	if c.Fs.EntryTimeout > 0 {
		err := c.validateDuration(c.Fs.EntryTimeout)
		if err != nil {
			return fmt.Errorf("invalid fs.entryTimeout: %w", err)
		}
	} else {
		c.Fs.EntryTimeout = configDefaultFsTimeout
	}

	return nil
}

func (c *Config) newDefaultFilesystem() *FilesystemConfig {
	return &FilesystemConfig{
		AllowOthers:  false,
		EntryTimeout: configDefaultFsTimeout,

		Debug: false,
	}
}

func (c *Config) validateDuration(d time.Duration) error {
	if d < configMinimalDuration {
		return fmt.Errorf("duration %v is too small, it should >= %v", d, configMinimalDuration)
	}
	if d > configMaximalDuration {
		return fmt.Errorf("duration %v is too big, it should <= %v", d, configMaximalDuration)
	}

	return nil
}

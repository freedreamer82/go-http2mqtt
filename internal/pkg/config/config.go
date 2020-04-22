package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	humanize "github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

type duration struct {
	time.Duration
}

type logLevel struct {
	logrus.Level
}

type fileSize struct {
	Size uint64
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func (l *logLevel) UnmarshalText(text []byte) error {
	var err error
	l.Level, err = logrus.ParseLevel(string(text))
	return err
}

func (s *fileSize) UnmarshalText(text []byte) error {
	var err error
	s.Size, err = humanize.ParseBytes(string(text))
	return err
}


type Config struct {
	LogEnabled bool
	LogPath    string
	LogLevel   logLevel
}

func Default() Config {
	return Config{
		LogEnabled: true,
		LogPath:    "",
		LogLevel:   logLevel{logrus.InfoLevel},
	}
}

func NewCopy(config *Config) Config {
	return Config{

		LogEnabled: config.LogEnabled,
		LogPath:    config.LogPath,
		LogLevel:   config.LogLevel,
	}
}

func Parse(filename string) (*Config, error) {
	resConfig := Default()

	var md toml.MetaData
	var err error
	md, err = toml.DecodeFile(filename, &resConfig)

	if err != nil {
		return nil, err
	}

	if len(md.Undecoded()) != 0 {
		err = fmt.Errorf("TOML is not formatted correctly! These parameters are unknown: %v", md.Undecoded())
		return nil, err
	}

	return &resConfig, nil
}

func (c *Config) PrintDebug(logger *logrus.Logger) {

}

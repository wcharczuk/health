package health

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultMaxStats is the default number of deltas to keep per host.
	DefaultMaxStats = 128
	// DefaultPingTimeout is the connection timeout.
	DefaultPingTimeout = 1500 * time.Millisecond
	//DefaultPollInterval is the default time between pings.
	DefaultPollInterval = 2000 * time.Millisecond

	// ExtensionJSON is the json extension.
	ExtensionJSON = ".json"
	// ExtensionYML is the json extension.
	ExtensionYML = ".yml"
	// ExtensionYAML is the json extension.
	ExtensionYAML = ".yaml"
)

// NewConfig returns a config with defaults.
func NewConfig() *Config {
	return &Config{
		PollInterval: DefaultPollInterval,
		PingTimeout:  DefaultPingTimeout,
		MaxStats:     DefaultMaxStats,
	}
}

// NewConfigFromFlags parses commandline flags into a config object.
func NewConfigFromFlags() (*Config, error) {
	var hosts HostsFlag
	flag.Var(&hosts, "host", "Host(s) to ping.")
	pollInterval := flag.Duration("interval", DefaultPollInterval, "Server polling interval as a duration")
	configFilePath := flag.String("config", "", "Load configuration from a file.")

	flag.Parse()

	if configFilePath != nil && len(*configFilePath) != 0 {
		return NewConfigFromPath(*configFilePath)
	}

	c := NewConfig()
	if pollInterval != nil {
		c.PollInterval = *pollInterval
	}
	if len(hosts) != 0 {
		c.Hosts = append(c.Hosts, hosts...)
	}

	return c, nil
}

// NewConfigFromPath returns a new config from a file path.
func NewConfigFromPath(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	filePathLower := strings.ToLower(filePath)

	config := NewConfig()

	if strings.HasSuffix(filePathLower, ExtensionJSON) {
		return config, json.NewDecoder(file).Decode(config)
	}

	if strings.HasSuffix(filePathLower, ExtensionYML) || strings.HasSuffix(filePathLower, ExtensionYAML) {
		contents, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return config, yaml.Unmarshal(contents, config)
	}
	return nil, fmt.Errorf("unsupported file type")
}

// Config is the healthcheck configuration.
type Config struct {
	PingTimeout  time.Duration `json:"ping_timeout" yaml:"pingTimeout"`
	MaxStats     int           `json:"max_stats" yaml:"maxStats"`
	PollInterval time.Duration `json:"interval" yaml:"pollInterval"`
	Hosts        []string      `json:"hosts" yaml:"hosts"`
	Verbose      bool          `json:"verbose" yaml:"verbose"`
}

// HostNameLength returns the length of the longest host name in the config.
func (c *Config) HostNameLength() int {
	longestHostName := 0
	for x := 0; x < len(c.Hosts); x++ {
		l := len(c.Hosts[x])
		if l > longestHostName {
			longestHostName = l
		}
	}
	return longestHostName
}

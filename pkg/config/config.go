// Package config handles application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds application configuration from environment variables.
type Config struct {
	ESHost        string
	ESIndex       string
	ESUsername    string
	ESPassword    string
	ReposPath     string
	GitOrg        string
	GitRepos      []string
	GitURLFormat  string
	IndexInterval time.Duration
	HTTPAddr      string
	LogLevel      string
	GitSSHKeyPath string
	GitToken      string
	Mode          string
}

// Load loads configuration from environment variables.
func Load() (cfg Config, err error) {
	cfg = Config{
		ESHost:        getEnv("ES_HOST", "http://localhost:9200"),
		ESIndex:       getEnv("ES_INDEX", "code-index"),
		ESUsername:    getEnv("ES_USERNAME", ""),
		ESPassword:    getEnv("ES_PASSWORD", ""),
		ReposPath:     getEnv("REPOS_PATH", "/repos"),
		GitOrg:        getEnv("GIT_ORG", ""),
		GitURLFormat:  getEnv("GIT_URL_TEMPLATE", "git@github.com:{org}/{repo}.git"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		GitSSHKeyPath: getEnv("GIT_SSH_KEY_PATH", ""),
		GitToken:      getEnv("GIT_TOKEN", ""),
	}

	intervalStr := getEnv("INDEX_INTERVAL", "5m")
	cfg.IndexInterval, err = time.ParseDuration(intervalStr)
	if err != nil {
		err = fmt.Errorf("invalid INDEX_INTERVAL: %w", err)
		return cfg, err
	}

	reposStr := getEnv("GIT_REPOS", "")
	if reposStr != "" {
		cfg.GitRepos = strings.Split(reposStr, ",")
		for i := range cfg.GitRepos {
			cfg.GitRepos[i] = strings.TrimSpace(cfg.GitRepos[i])
		}
	}

	return cfg, err
}

func getEnv(key string, defaultVal string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultVal
	}
	return value
}

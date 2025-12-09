package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{
				ESHost:        "http://localhost:9200",
				ESIndex:       "code-index",
				ESUsername:    "",
				ESPassword:    "",
				ReposPath:     "/repos",
				GitOrg:        "",
				GitRepos:      nil,
				GitURLFormat:  "git@github.com:{org}/{repo}.git",
				IndexInterval: 5 * time.Minute,
				HTTPAddr:      ":8080",
				LogLevel:      "info",
				GitSSHKeyPath: "",
				GitToken:      "",
			},
			wantErr: false,
		},
		{
			name: "custom values",
			env: map[string]string{
				"ES_HOST":          "http://es.example.com:9200",
				"ES_INDEX":         "my-code-index",
				"ES_USERNAME":      "elastic",
				"ES_PASSWORD":      "secret",
				"REPOS_PATH":       "/var/lib/repos",
				"GIT_ORG":          "myorg",
				"GIT_REPOS":        "repo1,repo2,repo3",
				"GIT_URL_TEMPLATE": "https://github.com/{org}/{repo}.git",
				"INDEX_INTERVAL":   "10m",
				"HTTP_ADDR":        ":9090",
				"LOG_LEVEL":        "debug",
				"GIT_SSH_KEY_PATH": "/keys/id_rsa",
				"GIT_TOKEN":        "ghp_token123",
			},
			want: Config{
				ESHost:        "http://es.example.com:9200",
				ESIndex:       "my-code-index",
				ESUsername:    "elastic",
				ESPassword:    "secret",
				ReposPath:     "/var/lib/repos",
				GitOrg:        "myorg",
				GitRepos:      []string{"repo1", "repo2", "repo3"},
				GitURLFormat:  "https://github.com/{org}/{repo}.git",
				IndexInterval: 10 * time.Minute,
				HTTPAddr:      ":9090",
				LogLevel:      "debug",
				GitSSHKeyPath: "/keys/id_rsa",
				GitToken:      "ghp_token123",
			},
			wantErr: false,
		},
		{
			name: "repos with whitespace",
			env: map[string]string{
				"GIT_REPOS": "repo1 , repo2,  repo3  ",
			},
			want: Config{
				ESHost:        "http://localhost:9200",
				ESIndex:       "code-index",
				ReposPath:     "/repos",
				GitRepos:      []string{"repo1", "repo2", "repo3"},
				GitURLFormat:  "git@github.com:{org}/{repo}.git",
				IndexInterval: 5 * time.Minute,
				HTTPAddr:      ":8080",
				LogLevel:      "info",
			},
			wantErr: false,
		},
		{
			name: "invalid interval",
			env: map[string]string{
				"INDEX_INTERVAL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "various duration formats",
			env: map[string]string{
				"INDEX_INTERVAL": "1h30m",
			},
			want: Config{
				ESHost:        "http://localhost:9200",
				ESIndex:       "code-index",
				ReposPath:     "/repos",
				GitURLFormat:  "git@github.com:{org}/{repo}.git",
				IndexInterval: 90 * time.Minute,
				HTTPAddr:      ":8080",
				LogLevel:      "info",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			assertConfigEqual(t, got, tt.want)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envVal     string
		want       string
	}{
		{
			name:       "env var set",
			key:        "TEST_VAR",
			defaultVal: "default",
			envVal:     "custom",
			want:       "custom",
		},
		{
			name:       "env var empty",
			key:        "TEST_VAR",
			defaultVal: "default",
			envVal:     "",
			want:       "default",
		},
		{
			name:       "env var not set",
			key:        "NONEXISTENT_VAR",
			defaultVal: "default",
			envVal:     "",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.key, tt.envVal)
			}

			got := getEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func assertConfigEqual(t *testing.T, got Config, want Config) {
	t.Helper()

	if got.ESHost != want.ESHost {
		t.Errorf("ESHost = %v, want %v", got.ESHost, want.ESHost)
	}
	if got.ESIndex != want.ESIndex {
		t.Errorf("ESIndex = %v, want %v", got.ESIndex, want.ESIndex)
	}
	if got.ESUsername != want.ESUsername {
		t.Errorf("ESUsername = %v, want %v", got.ESUsername, want.ESUsername)
	}
	if got.ReposPath != want.ReposPath {
		t.Errorf("ReposPath = %v, want %v", got.ReposPath, want.ReposPath)
	}
	if got.GitOrg != want.GitOrg {
		t.Errorf("GitOrg = %v, want %v", got.GitOrg, want.GitOrg)
	}
	if got.IndexInterval != want.IndexInterval {
		t.Errorf("IndexInterval = %v, want %v", got.IndexInterval, want.IndexInterval)
	}
	if got.HTTPAddr != want.HTTPAddr {
		t.Errorf("HTTPAddr = %v, want %v", got.HTTPAddr, want.HTTPAddr)
	}

	assertGitReposEqual(t, got.GitRepos, want.GitRepos)
}

func assertGitReposEqual(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("GitRepos length = %v, want %v", len(got), len(want))
		return
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("GitRepos[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"ES_HOST",
		"ES_INDEX",
		"ES_USERNAME",
		"ES_PASSWORD",
		"REPOS_PATH",
		"GIT_ORG",
		"GIT_REPOS",
		"GIT_URL_TEMPLATE",
		"INDEX_INTERVAL",
		"HTTP_ADDR",
		"LOG_LEVEL",
		"GIT_SSH_KEY_PATH",
		"GIT_TOKEN",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

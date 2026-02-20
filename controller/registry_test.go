package controller

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryConfig(t *testing.T) {
	cfg, err := NewRegistryConfig("https://ghcr.io/coroot", "", false)
	require.NoError(t, err)
	assert.Equal(t, "ghcr.io/coroot", cfg.ImagePrefix)
	assert.Equal(t, "ghcr.io", cfg.host)

	cfg, err = NewRegistryConfig("https://artifactory.example.com/docker-local/coroot", "", false)
	require.NoError(t, err)
	assert.Equal(t, "artifactory.example.com/docker-local/coroot", cfg.ImagePrefix)
	assert.Equal(t, "artifactory.example.com", cfg.host)

	cfg, err = NewRegistryConfig("https://registry.example.com:8443/coroot", "", false)
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com:8443/coroot", cfg.ImagePrefix)
	assert.Equal(t, "registry.example.com", cfg.host)

	cfg, err = NewRegistryConfig("https://registry.example.com", "", false)
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com", cfg.ImagePrefix)
	assert.Equal(t, "registry.example.com", cfg.host)

	// empty URL defaults to ghcr.io/coroot
	cfg, err = NewRegistryConfig("", "", false)
	require.NoError(t, err)
	assert.Equal(t, "ghcr.io/coroot", cfg.ImagePrefix)
}

func dockerConfigJSON(t *testing.T, auths map[string]dockerConfigAuth) []byte {
	t.Helper()
	data, err := json.Marshal(dockerConfig{Auths: auths})
	require.NoError(t, err)
	return data
}

func TestLoadDockerConfig(t *testing.T) {
	// username and password fields
	cfg := &RegistryConfig{host: "artifactory.example.com"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"https://artifactory.example.com": {Username: "user", Password: "pass"},
	})))
	assert.Equal(t, "user", cfg.username)
	assert.Equal(t, "pass", cfg.password)

	// base64 auth field
	cfg = &RegistryConfig{host: "ghcr.io"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"ghcr.io": {Auth: base64.StdEncoding.EncodeToString([]byte("myuser:mypass"))},
	})))
	assert.Equal(t, "myuser", cfg.username)
	assert.Equal(t, "mypass", cfg.password)

	// host with port
	cfg = &RegistryConfig{host: "registry.example.com"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"https://registry.example.com:8443": {Username: "admin", Password: "secret"},
	})))
	assert.Equal(t, "admin", cfg.username)
	assert.Equal(t, "secret", cfg.password)

	// no matching host
	cfg = &RegistryConfig{host: "ghcr.io"}
	assert.Error(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"docker.io": {Username: "user", Password: "pass"},
	})))

	// host without scheme in auths
	cfg = &RegistryConfig{host: "ghcr.io"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"ghcr.io": {Username: "user", Password: "pass"},
	})))
	assert.Equal(t, "user", cfg.username)
	assert.Equal(t, "pass", cfg.password)

	// URL with path in auths (e.g. Docker Hub style)
	cfg = &RegistryConfig{host: "index.docker.io"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"https://index.docker.io/v1/": {Auth: base64.StdEncoding.EncodeToString([]byte("dockeruser:dockerpass"))},
	})))
	assert.Equal(t, "dockeruser", cfg.username)
	assert.Equal(t, "dockerpass", cfg.password)

	// multiple entries, correct one is matched
	cfg = &RegistryConfig{host: "registry.example.com"}
	require.NoError(t, cfg.loadDockerConfig(dockerConfigJSON(t, map[string]dockerConfigAuth{
		"https://index.docker.io/v1/": {Auth: base64.StdEncoding.EncodeToString([]byte("dockeruser:dockerpass"))},
		"ghcr.io":                     {Auth: base64.StdEncoding.EncodeToString([]byte("ghuser:ghtoken"))},
		"registry.example.com":        {Username: "myuser", Password: "mypassword"},
	})))
	assert.Equal(t, "myuser", cfg.username)
	assert.Equal(t, "mypassword", cfg.password)
}

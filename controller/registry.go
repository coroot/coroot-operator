package controller

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	DefaultImageRegistry = "https://ghcr.io/coroot"

	dockerConfigPath = "/etc/registry/config.json"
)

type RegistryConfig struct {
	// ImagePrefix is the scheme-stripped registry path used in image references,
	// e.g. "ghcr.io/coroot" or "artifactory.example.com/docker-local/coroot".
	ImagePrefix    string
	PullSecretName string
	TLSSkipVerify  bool

	host     string
	username string
	password string
}

type dockerConfig struct {
	Auths map[string]dockerConfigAuth `json:"auths"`
}

type dockerConfigAuth struct {
	Auth     string `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewRegistryConfig(registryURL, pullSecret string, tlsSkipVerify bool) (*RegistryConfig, error) {
	if registryURL == "" {
		registryURL = DefaultImageRegistry
	}
	u, err := url.Parse(registryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid registry URL %q: %w", registryURL, err)
	}
	cfg := &RegistryConfig{
		PullSecretName: pullSecret,
		TLSSkipVerify:  tlsSkipVerify,
		host:           u.Hostname(),
		ImagePrefix:    u.Host + u.Path,
	}

	if cfg.PullSecretName != "" {
		if err := cfg.loadDockerConfigFrom(dockerConfigPath); err != nil {
			return nil, fmt.Errorf("failed to load registry credentials from %s: %w", dockerConfigPath, err)
		}
	}
	return cfg, nil
}

func (c *RegistryConfig) loadDockerConfigFrom(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return c.loadDockerConfig(data)
}

func (c *RegistryConfig) loadDockerConfig(data []byte) error {
	var dc dockerConfig
	if err := json.Unmarshal(data, &dc); err != nil {
		return fmt.Errorf("failed to parse docker config: %w", err)
	}

	for host, auth := range dc.Auths {
		h := host
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			h = u.Hostname()
		} else {
			h = strings.TrimSuffix(h, "/")
		}
		if c.host != h {
			continue
		}
		if auth.Username != "" && auth.Password != "" {
			c.username = auth.Username
			c.password = auth.Password
			return nil
		}
		if auth.Auth != "" {
			decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
			if err != nil {
				return fmt.Errorf("failed to decode auth field: %w", err)
			}
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				c.username = parts[0]
				c.password = parts[1]
				return nil
			}
		}
	}
	return fmt.Errorf("no credentials found for registry host %q", c.host)
}

func (c *RegistryConfig) Image(name string) string {
	return c.ImagePrefix + "/" + name
}

func (c *RegistryConfig) RemoteOptions() []remote.Option {
	var opts []remote.Option
	if c.username != "" && c.password != "" {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: c.username,
			Password: c.password,
		}))
	}
	if c.TLSSkipVerify {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		opts = append(opts, remote.WithTransport(transport))
	}
	return opts
}

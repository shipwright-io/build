package config

import (
	"os"
	"strconv"
	"time"
)

const (
	contextTimeout = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contexTimeoutEnvVar = "CTX_TIMEOUT"

	kanikoDefaultImage = "gcr.io/kaniko-project/executor:v0.24.0"
	// kanikoImageEnvVar environment variable for Kaniko container image, for instance:
	// KANIKO_CONTAINER_IMAGE="gcr.io/kaniko-project/executor:v0.24.0"
	kanikoImageEnvVar = "KANIKO_CONTAINER_IMAGE"
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut           time.Duration
	KanikoContainerImage string
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut:           contextTimeout,
		KanikoContainerImage: kanikoDefaultImage,
	}
}

// SetConfigFromEnv updates the configuration managed by environment variables.
func (c *Config) SetConfigFromEnv() error {
	timeout := os.Getenv(contexTimeoutEnvVar)
	if timeout != "" {
		i, err := strconv.Atoi(timeout)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}

	kanikoImage := os.Getenv(kanikoImageEnvVar)
	if kanikoImage != "" {
		c.KanikoContainerImage = kanikoImage
	}
	return nil
}

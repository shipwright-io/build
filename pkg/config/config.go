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
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut time.Duration
}

// NewDefaultConfig returns a new Config
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut: contextTimeout,
	}
}

// SetConfigTimeOutFromEnv updates the CtxTimeOut field
// by consuming the value from an ENV variable.
func (c *Config) SetConfigTimeOutFromEnv() error {
	timeOut := os.Getenv(contexTimeoutEnvVar)
	if timeOut != "" {
		i, err := strconv.Atoi(timeOut)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}
	return nil
}

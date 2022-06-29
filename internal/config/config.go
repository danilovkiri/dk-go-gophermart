// Package config provides types for handling configuration parameters.
package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

// Config handles server-related constants and parameters.
type Config struct {
	ServerConfig  *ServerConfig
	StorageConfig *StorageConfig
	SecretConfig  *SecretConfig
	QueueConfig   *QueueConfig
}

// QueueConfig defines default parallelization parameters for queue.
type QueueConfig struct {
	WorkerNumber int `env:"N_WORKERS"`
}

// ServerConfig defines default server-relates constants and parameters and overwrites them with environment variables.
type ServerConfig struct {
	ServerAddress  string `env:"RUN_ADDRESS"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

// StorageConfig retrieves file inpsql-related parameters from environment.
type StorageConfig struct {
	DatabaseDSN string `env:"DATABASE_URI"`
}

// SecretConfig retrieves a secret user key for hashing.
type SecretConfig struct {
	SecretKey string `env:"SECRET_KEY" envDefault:"jds__63h3_7ds"`
}

// NewQueueConfig sets up a queueing configuration.
func NewQueueConfig() (*QueueConfig, error) {
	cfg := QueueConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewStorageConfig sets up a inpsql configuration.
func NewStorageConfig() (*StorageConfig, error) {
	cfg := StorageConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewServerConfig sets up a server configuration.
func NewServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewSecretConfig sets up a secret configuration.
func NewSecretConfig() (*SecretConfig, error) {
	cfg := SecretConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewConfiguration sets up a total configuration.
func NewConfiguration() (*Config, error) {
	queueCfg, err := NewQueueConfig()
	if err != nil {
		return nil, err
	}
	serverCfg, err := NewServerConfig()
	if err != nil {
		return nil, err
	}
	storageCfg, err := NewStorageConfig()
	if err != nil {
		return nil, err
	}
	secretConfig, err := NewSecretConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		ServerConfig:  serverCfg,
		StorageConfig: storageCfg,
		SecretConfig:  secretConfig,
		QueueConfig:   queueCfg,
	}, nil
}

// isFlagPassed checks whether the flag was set in CLI
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// ParseFlags parses command line arguments and stores them
func (c *Config) ParseFlags() {
	a := flag.String("a", ":8080", "Server address")
	r := flag.String("r", "http://localhost:7070", "Accrual service address")
	// DatabaseDSN scheme: "postgres://username:password@localhost:5432/database_name"
	d := flag.String("d", "", "PSQL DB connection DSN")
	n := flag.Int("n", 7, "Number of additional workers (1 worker will still be )")
	flag.Parse()
	// priority: flag -> env -> default flag
	// note that env parsing precedes flag parsing
	if isFlagPassed("a") || c.ServerConfig.ServerAddress == "" {
		c.ServerConfig.ServerAddress = *a
	}
	if isFlagPassed("r") || c.ServerConfig.AccrualAddress == "" {
		c.ServerConfig.AccrualAddress = *r
	}
	if isFlagPassed("d") || c.StorageConfig.DatabaseDSN == "" {
		c.StorageConfig.DatabaseDSN = *d
	}
	if isFlagPassed("n") || c.QueueConfig.WorkerNumber == 0 {
		c.QueueConfig.WorkerNumber = *n
		if c.QueueConfig.WorkerNumber <= 0 {
			log.Panic("Number of workers must be a non-negative integer")
		}
	}
}

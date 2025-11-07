package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPHost        = "127.0.0.1"
	defaultGRPCHost        = "127.0.0.1"
	defaultHTTPPort        = 8087
	defaultGRPCPort        = 50055
	defaultShutdownSeconds = 5
)

type Config struct {
	HTTPAddr      string
	GRPCAddr      string
	ShutdownGrace time.Duration
}

func Load() Config {
	httpAddr := joinHostPort(lookupEnv("HTTP_HOST", defaultHTTPHost), lookupEnvInt("HTTP_PORT", defaultHTTPPort))
	grpcAddr := joinHostPort(lookupEnv("GRPC_HOST", defaultGRPCHost), lookupEnvInt("GRPC_PORT", defaultGRPCPort))
	grace := time.Duration(lookupEnvInt("SHUTDOWN_GRACE_SECONDS", defaultShutdownSeconds)) * time.Second

	return Config{
		HTTPAddr:      httpAddr,
		GRPCAddr:      grpcAddr,
		ShutdownGrace: grace,
	}
}

func lookupEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func lookupEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func joinHostPort(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

package config

import (
	"os"
	"strconv"
)

const (
	CONFIG_KEY_PORT                    				= "YTCLIPPER_PORT"
	CONFIG_KEY_DEBUG                    			= "YTCLIPPER_DEBUG"
	CONFIG_KEY_CLIP_SIZE_LIMIT_IN_MB         		= "YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB"
	
	CONFIG_KEY_YT_DLP_PROXY							= "YTCLIPPER_YT_DLP_PROXY"
	CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS	= "YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS"

	CONFIG_KEY_RATE_LIMITER_RATE       				= "YTCLIPPER_RATE_LIMITER_RATE"
	CONFIG_KEY_RATE_LIMITER_BURST      				= "YTCLIPPER_RATE_LIMITER_BURST"
	CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES		= "YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES"

	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED"
)

var CONFIG *Config = NewConfig()

type Config struct {
	Port			string
	ClipSizeInMb 	int64
	Debug			bool
	RateLimiterConfig RateLimiterConfig
	YtDlpConfig YtDlpConfig
	ClipCleanUpSchedulerConfig ClipCleanUpSchedulerConfig
}

type RateLimiterConfig struct {
	Rate 	float64
	Burst	int
	ExpiresInMinutes int
}

type YtDlpConfig struct {
	CommandTimeoutInSeconds int
	Proxy string
}

type ClipCleanUpSchedulerConfig struct {
	IntervalInMinutes int
	ClipDirectoryPath string
	IsEnabled bool
}

func NewClipCleanUpSchedulerConfig() *ClipCleanUpSchedulerConfig {
	intervalInMinutes := GetEnvInt(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES, 5)
	clipDirectoryPath := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH, "./videos")
	clipSchedulerEnabled := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED, "true") == "true"

	return &ClipCleanUpSchedulerConfig{
		IntervalInMinutes: intervalInMinutes,
		ClipDirectoryPath: clipDirectoryPath,
		IsEnabled: clipSchedulerEnabled,
	}
}

func NewYtDlpConfig() *YtDlpConfig{
	commandTimeoutInSeconds := GetEnvInt(CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS, 60)
	proxy := GetEnv(CONFIG_KEY_YT_DLP_PROXY, "")

	return &YtDlpConfig{
		CommandTimeoutInSeconds: commandTimeoutInSeconds,
		Proxy: proxy,
	}
} 

func NewRateLimiterConfig() *RateLimiterConfig {
	rate := GetEnvInt(CONFIG_KEY_RATE_LIMITER_RATE, 5)
	burst := GetEnvInt(CONFIG_KEY_RATE_LIMITER_BURST, 20)
	expiresInMinutes := GetEnvInt(CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES, 1)

	return &RateLimiterConfig{
		Rate: 			  float64(rate),
		Burst:            burst,
		ExpiresInMinutes: expiresInMinutes,
	}
}

func NewConfig() *Config {
	port := GetEnv(CONFIG_KEY_PORT, "8080")

	debug := GetEnv(CONFIG_KEY_DEBUG, "true") == "true"
	clipSizeInMb := GetClipSizeLimit()

	return &Config{
		Port:  port,
		Debug: debug,
		ClipSizeInMb: clipSizeInMb,
		RateLimiterConfig:  *NewRateLimiterConfig(),
		YtDlpConfig:*NewYtDlpConfig(),
		ClipCleanUpSchedulerConfig: *NewClipCleanUpSchedulerConfig(),
	}
}

func GetEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	intValue, err:=strconv.Atoi(value)
	if err != nil{
		return fallback
	}
	return intValue
}

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func GetClipSizeLimit() int64 {
	clipSizeLimitBytes := int64(GetEnvInt(CONFIG_KEY_CLIP_SIZE_LIMIT_IN_MB, 300)) * 1024 * 1024
	return clipSizeLimitBytes
}


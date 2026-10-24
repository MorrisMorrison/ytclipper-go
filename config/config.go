package config

import (
	"os"
	"strconv"
	"strings"
	"ytclipper-go/utils"
)

const (
	CONFIG_KEY_PORT                               = "YTCLIPPER_PORT"
	CONFIG_KEY_DEBUG                              = "YTCLIPPER_DEBUG"
	CONFIG_KEY_YT_DLP_CLIP_SIZE_LIMIT_IN_MB       = "YTCLIPPER_YT_DLP_CLIP_SIZE_LIMIT_IN_MB"
	CONFIG_KEY_YT_DLP_PROXY                       = "YTCLIPPER_YT_DLP_PROXY"
	CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS  = "YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS"
	CONFIG_KEY_YT_DLP_DOWNLOAD_TIMEOUT_IN_SECONDS = "YTCLIPPER_YT_DLP_DOWNLOAD_TIMEOUT_IN_SECONDS"
	CONFIG_KEY_YT_DLP_EXTRACTOR_RETRIES           = "YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES"

	CONFIG_KEY_RATE_LIMITER_RATE               = "YTCLIPPER_RATE_LIMITER_RATE"
	CONFIG_KEY_RATE_LIMITER_BURST              = "YTCLIPPER_RATE_LIMITER_BURST"
	CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES = "YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES"
	CONFIG_KEY_TRUSTED_PROXIES                 = "YTCLIPPER_TRUSTED_PROXIES"

	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES  = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH  = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED              = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_RETENTION_IN_MINUTES = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_RETENTION_IN_MINUTES"

	CONFIG_KEY_MAX_CONCURRENT_JOBS        = "YTCLIPPER_MAX_CONCURRENT_JOBS"
	CONFIG_KEY_MAX_QUEUED_JOBS            = "YTCLIPPER_MAX_QUEUED_JOBS"
	CONFIG_KEY_MAX_CLIP_LENGTH_IN_MINUTES = "YTCLIPPER_MAX_CLIP_LENGTH_IN_MINUTES"

	CONFIG_KEY_AUTH_USERNAME = "YTCLIPPER_AUTH_USERNAME"
	CONFIG_KEY_AUTH_PASSWORD = "YTCLIPPER_AUTH_PASSWORD"
)

var CONFIG *Config = NewConfig()

type Config struct {
	Port              string
	Debug             bool
	MaxConcurrentJobs int
	// MaxQueuedJobs caps queued plus processing jobs; clip requests beyond it
	// are rejected with 429.
	MaxQueuedJobs int
	// MaxClipLengthInMinutes caps the requested clip range; 0 disables it.
	MaxClipLengthInMinutes int
	// TrustedProxies are the reverse proxies (IPs or CIDRs) whose
	// X-Forwarded-For header is believed. Empty: the client IP is the remote
	// address of the connection.
	TrustedProxies             []string
	RateLimiterConfig          RateLimiterConfig
	YtDlpConfig                YtDlpConfig
	ClipCleanUpSchedulerConfig ClipCleanUpSchedulerConfig
	BasicAuthConfig            BasicAuthConfig
}

type RateLimiterConfig struct {
	Rate             float64
	Burst            int
	ExpiresInMinutes int
}

type YtDlpConfig struct {
	ClipSizeInMb             int64 // in bytes, despite the name
	CommandTimeoutInSeconds  int   // yt-dlp -J (video info)
	DownloadTimeoutInSeconds int   // one clip download, yt-dlp and ffmpeg together
	Proxy                    string
	ExtractorRetries         int
}

type ClipCleanUpSchedulerConfig struct {
	IntervalInMinutes  int
	RetentionInMinutes int
	ClipDirectoryPath  string
	IsEnabled          bool
}

type BasicAuthConfig struct {
	Username string
	Password string
}

func NewClipCleanUpSchedulerConfig() *ClipCleanUpSchedulerConfig {
	intervalInMinutes := GetEnvInt(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES, 5)
	retentionInMinutes := GetEnvInt(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_RETENTION_IN_MINUTES, 30)
	clipDirectoryPath := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH, "./videos/")
	clipSchedulerEnabled := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED, "true") == "true"

	return &ClipCleanUpSchedulerConfig{
		IntervalInMinutes:  intervalInMinutes,
		RetentionInMinutes: retentionInMinutes,
		ClipDirectoryPath:  clipDirectoryPath,
		IsEnabled:          clipSchedulerEnabled,
	}
}

func NewYtDlpConfig() *YtDlpConfig {
	clipSizeInMb := utils.MbToBytes(GetEnvInt(CONFIG_KEY_YT_DLP_CLIP_SIZE_LIMIT_IN_MB, 300))
	commandTimeoutInSeconds := GetEnvInt(CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS, 60)
	downloadTimeoutInSeconds := GetEnvInt(CONFIG_KEY_YT_DLP_DOWNLOAD_TIMEOUT_IN_SECONDS, 600)
	proxy := GetEnv(CONFIG_KEY_YT_DLP_PROXY, "")
	extractorRetries := GetEnvInt(CONFIG_KEY_YT_DLP_EXTRACTOR_RETRIES, 3)

	return &YtDlpConfig{
		ClipSizeInMb:             clipSizeInMb,
		CommandTimeoutInSeconds:  commandTimeoutInSeconds,
		DownloadTimeoutInSeconds: downloadTimeoutInSeconds,
		Proxy:                    proxy,
		ExtractorRetries:         extractorRetries,
	}
}

func NewRateLimiterConfig() *RateLimiterConfig {
	rate := GetEnvInt(CONFIG_KEY_RATE_LIMITER_RATE, 5)
	burst := GetEnvInt(CONFIG_KEY_RATE_LIMITER_BURST, 20)
	expiresInMinutes := GetEnvInt(CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES, 1)

	return &RateLimiterConfig{
		Rate:             float64(rate),
		Burst:            burst,
		ExpiresInMinutes: expiresInMinutes,
	}
}

func NewBasicAuthConfig() *BasicAuthConfig {
	username := GetEnv(CONFIG_KEY_AUTH_USERNAME, "")
	password := GetEnv(CONFIG_KEY_AUTH_PASSWORD, "")

	return &BasicAuthConfig{
		Username: username,
		Password: password,
	}
}

func NewConfig() *Config {
	port := GetEnv(CONFIG_KEY_PORT, "8080")
	debug := GetEnv(CONFIG_KEY_DEBUG, "false") == "true"
	maxConcurrentJobs := max(1, GetEnvInt(CONFIG_KEY_MAX_CONCURRENT_JOBS, 3))

	// Unset (or not positive): four jobs per worker. Never fewer than the
	// workers themselves.
	maxQueuedJobs := GetEnvInt(CONFIG_KEY_MAX_QUEUED_JOBS, 0)
	if maxQueuedJobs <= 0 {
		maxQueuedJobs = 4 * maxConcurrentJobs
	}

	return &Config{
		Port:                       port,
		Debug:                      debug,
		MaxConcurrentJobs:          maxConcurrentJobs,
		MaxQueuedJobs:              max(maxQueuedJobs, maxConcurrentJobs),
		MaxClipLengthInMinutes:     max(0, GetEnvInt(CONFIG_KEY_MAX_CLIP_LENGTH_IN_MINUTES, 30)),
		TrustedProxies:             GetEnvList(CONFIG_KEY_TRUSTED_PROXIES),
		RateLimiterConfig:          *NewRateLimiterConfig(),
		YtDlpConfig:                *NewYtDlpConfig(),
		ClipCleanUpSchedulerConfig: *NewClipCleanUpSchedulerConfig(),
		BasicAuthConfig:            *NewBasicAuthConfig(),
	}
}

func GetEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return intValue
}

// GetEnvList splits a comma-separated variable into its trimmed, non-empty
// entries.
func GetEnvList(key string) []string {
	var list []string
	for _, v := range strings.Split(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			list = append(list, v)
		}
	}
	return list
}

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

package config

import (
	"os"
	"strconv"
	"ytclipper-go/utils"
)

const (
	CONFIG_KEY_PORT                              = "YTCLIPPER_PORT"
	CONFIG_KEY_DEBUG                             = "YTCLIPPER_DEBUG"
	CONFIG_KEY_YT_DLP_CLIP_SIZE_LIMIT_IN_MB      = "YTCLIPPER_YT_DLP_CLIP_SIZE_LIMIT_IN_MB"
	CONFIG_KEY_YT_DLP_PROXY                      = "YTCLIPPER_YT_DLP_PROXY"
	CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS = "YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS"
	CONFIG_KEY_YT_DLP_COOKIES_FILE               = "YTCLIPPER_YT_DLP_COOKIES_FILE"
	CONFIG_KEY_YT_DLP_COOKIES_CONTENT            = "YTCLIPPER_YT_DLP_COOKIES_CONTENT"
	CONFIG_KEY_YT_DLP_USER_AGENT                 = "YTCLIPPER_YT_DLP_USER_AGENT"
	CONFIG_KEY_YT_DLP_EXTRACTOR_RETRIES          = "YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES"
	CONFIG_KEY_YT_DLP_SLEEP_INTERVAL             = "YTCLIPPER_YT_DLP_SLEEP_INTERVAL"
	CONFIG_KEY_YT_DLP_ENABLE_USER_AGENT_ROTATION = "YTCLIPPER_YT_DLP_ENABLE_USER_AGENT_ROTATION"

	CONFIG_KEY_RATE_LIMITER_RATE               = "YTCLIPPER_RATE_LIMITER_RATE"
	CONFIG_KEY_RATE_LIMITER_BURST              = "YTCLIPPER_RATE_LIMITER_BURST"
	CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES = "YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES"

	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH"
	CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED             = "YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED"

	CONFIG_KEY_COOKIE_MONITOR_ENABLED             = "YTCLIPPER_COOKIE_MONITOR_ENABLED"
	CONFIG_KEY_COOKIE_MONITOR_INTERVAL_HOURS      = "YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS"
	CONFIG_KEY_COOKIE_MONITOR_WARNING_THRESHOLD   = "YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS"
	CONFIG_KEY_COOKIE_MONITOR_URGENT_THRESHOLD    = "YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS"
	CONFIG_KEY_COOKIE_MONITOR_TOPIC               = "YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC"

	CONFIG_KEY_NTFY_ENABLED    = "YTCLIPPER_NTFY_ENABLED"
	CONFIG_KEY_NTFY_SERVER_URL = "YTCLIPPER_NTFY_SERVER_URL"

	CONFIG_KEY_AUTH_USERNAME = "YTCLIPPER_AUTH_USERNAME"
	CONFIG_KEY_AUTH_PASSWORD = "YTCLIPPER_AUTH_PASSWORD"
)

var CONFIG *Config = NewConfig()

type Config struct {
	Port                       string
	Debug                      bool
	RateLimiterConfig          RateLimiterConfig
	YtDlpConfig                YtDlpConfig
	ClipCleanUpSchedulerConfig ClipCleanUpSchedulerConfig
	CookieMonitorConfig        CookieMonitorConfig
	NtfyConfig                 NtfyConfig
	BasicAuthConfig            BasicAuthConfig
}

type RateLimiterConfig struct {
	Rate             float64
	Burst            int
	ExpiresInMinutes int
}

type YtDlpConfig struct {
	ClipSizeInMb            int64
	CommandTimeoutInSeconds int
	Proxy                   string
	CookiesFile             string
	CookiesContent          string
	UserAgent               string
	ExtractorRetries        int
	SleepInterval           int
	EnableUserAgentRotation bool
}

type ClipCleanUpSchedulerConfig struct {
	IntervalInMinutes int
	ClipDirectoryPath string
	IsEnabled         bool
}

type CookieMonitorConfig struct {
	Enabled              bool
	IntervalHours        int
	WarningThresholdDays int
	UrgentThresholdDays  int
	NtfyTopic            string
}

type NtfyConfig struct {
	Enabled   bool
	ServerURL string
}

type BasicAuthConfig struct {
	Username string
	Password string
}

func NewClipCleanUpSchedulerConfig() *ClipCleanUpSchedulerConfig {
	intervalInMinutes := GetEnvInt(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES, 5)
	clipDirectoryPath := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH, "./videos/")
	clipSchedulerEnabled := GetEnv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED, "true") == "true"

	return &ClipCleanUpSchedulerConfig{
		IntervalInMinutes: intervalInMinutes,
		ClipDirectoryPath: clipDirectoryPath,
		IsEnabled:         clipSchedulerEnabled,
	}
}

func NewYtDlpConfig() *YtDlpConfig {
	clipSizeInMb := utils.MbToBytes(GetEnvInt(CONFIG_KEY_YT_DLP_CLIP_SIZE_LIMIT_IN_MB, 300))
	commandTimeoutInSeconds := GetEnvInt(CONFIG_KEY_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS, 60)
	proxy := GetEnv(CONFIG_KEY_YT_DLP_PROXY, "")
	cookiesFile := GetEnv(CONFIG_KEY_YT_DLP_COOKIES_FILE, "")
	cookiesContent := GetEnv(CONFIG_KEY_YT_DLP_COOKIES_CONTENT, "")
	userAgent := GetEnv(CONFIG_KEY_YT_DLP_USER_AGENT, "")
	extractorRetries := GetEnvInt(CONFIG_KEY_YT_DLP_EXTRACTOR_RETRIES, 3)
	sleepInterval := GetEnvInt(CONFIG_KEY_YT_DLP_SLEEP_INTERVAL, 2)
	enableUserAgentRotation := GetEnv(CONFIG_KEY_YT_DLP_ENABLE_USER_AGENT_ROTATION, "true") == "true"

	return &YtDlpConfig{
		ClipSizeInMb:            clipSizeInMb,
		CommandTimeoutInSeconds: commandTimeoutInSeconds,
		Proxy:                   proxy,
		CookiesFile:             cookiesFile,
		CookiesContent:          cookiesContent,
		UserAgent:               userAgent,
		ExtractorRetries:        extractorRetries,
		SleepInterval:           sleepInterval,
		EnableUserAgentRotation: enableUserAgentRotation,
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

func NewCookieMonitorConfig() *CookieMonitorConfig {
	enabled := GetEnv(CONFIG_KEY_COOKIE_MONITOR_ENABLED, "true") == "true"
	intervalHours := GetEnvInt(CONFIG_KEY_COOKIE_MONITOR_INTERVAL_HOURS, 24)
	warningThresholdDays := GetEnvInt(CONFIG_KEY_COOKIE_MONITOR_WARNING_THRESHOLD, 30)
	urgentThresholdDays := GetEnvInt(CONFIG_KEY_COOKIE_MONITOR_URGENT_THRESHOLD, 7)
	ntfyTopic := GetEnv(CONFIG_KEY_COOKIE_MONITOR_TOPIC, "ytclipper-cookies")

	return &CookieMonitorConfig{
		Enabled:              enabled,
		IntervalHours:        intervalHours,
		WarningThresholdDays: warningThresholdDays,
		UrgentThresholdDays:  urgentThresholdDays,
		NtfyTopic:            ntfyTopic,
	}
}

func NewNtfyConfig() *NtfyConfig {
	enabled := GetEnv(CONFIG_KEY_NTFY_ENABLED, "false") == "true"
	serverURL := GetEnv(CONFIG_KEY_NTFY_SERVER_URL, "")

	return &NtfyConfig{
		Enabled:   enabled,
		ServerURL: serverURL,
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
	debug := GetEnv(CONFIG_KEY_DEBUG, "true") == "true"

	return &Config{
		Port:                       port,
		Debug:                      debug,
		RateLimiterConfig:          *NewRateLimiterConfig(),
		YtDlpConfig:                *NewYtDlpConfig(),
		ClipCleanUpSchedulerConfig: *NewClipCleanUpSchedulerConfig(),
		CookieMonitorConfig:        *NewCookieMonitorConfig(),
		NtfyConfig:                 *NewNtfyConfig(),
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

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

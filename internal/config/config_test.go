package config

import (
	"slices"
	"testing"
)

func TestNewConfigDefaults(t *testing.T) {
	for _, key := range []string{
		CONFIG_KEY_PORT, CONFIG_KEY_DEBUG, CONFIG_KEY_MAX_CONCURRENT_JOBS, CONFIG_KEY_MAX_QUEUED_JOBS,
		CONFIG_KEY_MAX_CLIP_LENGTH_IN_MINUTES, CONFIG_KEY_TRUSTED_PROXIES, CONFIG_KEY_YT_DLP_CLIP_SIZE_LIMIT_IN_MB,
		CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED, CONFIG_KEY_LOG_FORMAT,
	} {
		t.Setenv(key, "")
	}

	c := NewConfig()
	if c.Port != "8080" || c.Debug || c.LogFormat != "text" {
		t.Errorf("port/debug/log format = %q/%v/%q", c.Port, c.Debug, c.LogFormat)
	}
	if c.MaxConcurrentJobs != 3 || c.MaxQueuedJobs != 12 || c.MaxClipLengthInMinutes != 30 {
		t.Errorf("jobs/queue/clip length = %d/%d/%d; want 3/12/30", c.MaxConcurrentJobs, c.MaxQueuedJobs, c.MaxClipLengthInMinutes)
	}
	if c.YtDlpConfig.ClipSizeInMb != 300*1024*1024 {
		t.Errorf("clip size limit = %d bytes; want 300 MiB", c.YtDlpConfig.ClipSizeInMb)
	}
	if !c.ClipCleanUpSchedulerConfig.IsEnabled || c.ClipCleanUpSchedulerConfig.RetentionInMinutes != 30 {
		t.Errorf("cleanup = %+v", c.ClipCleanUpSchedulerConfig)
	}
	if c.TrustedProxies != nil {
		t.Errorf("trusted proxies = %v; want none", c.TrustedProxies)
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	t.Setenv(CONFIG_KEY_DEBUG, "true")
	t.Setenv(CONFIG_KEY_MAX_CONCURRENT_JOBS, "2")
	t.Setenv(CONFIG_KEY_MAX_QUEUED_JOBS, "1") // below the workers: raised to 2
	t.Setenv(CONFIG_KEY_MAX_CLIP_LENGTH_IN_MINUTES, "-5")
	t.Setenv(CONFIG_KEY_TRUSTED_PROXIES, " 10.0.0.0/8, ,192.0.2.10 ")
	t.Setenv(CONFIG_KEY_CLIP_CLEANUP_SCHEDULER_ENABLED, "false")
	t.Setenv(CONFIG_KEY_AUTH_USERNAME, "user")
	t.Setenv(CONFIG_KEY_AUTH_PASSWORD, "pass")
	t.Setenv(CONFIG_KEY_RATE_LIMITER_RATE, "9")

	c := NewConfig()
	if !c.Debug || c.MaxConcurrentJobs != 2 || c.MaxQueuedJobs != 2 || c.MaxClipLengthInMinutes != 0 {
		t.Errorf("debug/jobs/queue/clip length = %v/%d/%d/%d", c.Debug, c.MaxConcurrentJobs, c.MaxQueuedJobs, c.MaxClipLengthInMinutes)
	}
	if want := []string{"10.0.0.0/8", "192.0.2.10"}; !slices.Equal(c.TrustedProxies, want) {
		t.Errorf("trusted proxies = %v; want %v", c.TrustedProxies, want)
	}
	if c.ClipCleanUpSchedulerConfig.IsEnabled {
		t.Error("cleanup should be disabled")
	}
	if c.BasicAuthConfig.Username != "user" || c.BasicAuthConfig.Password != "pass" || c.RateLimiterConfig.Rate != 9 {
		t.Errorf("auth/rate = %+v/%v", c.BasicAuthConfig, c.RateLimiterConfig.Rate)
	}
}

func TestMaxConcurrentJobsIsAtLeastOne(t *testing.T) {
	t.Setenv(CONFIG_KEY_MAX_CONCURRENT_JOBS, "0")
	t.Setenv(CONFIG_KEY_MAX_QUEUED_JOBS, "")
	if c := NewConfig(); c.MaxConcurrentJobs != 1 || c.MaxQueuedJobs != 4 {
		t.Errorf("jobs/queue = %d/%d; want 1/4", c.MaxConcurrentJobs, c.MaxQueuedJobs)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("YTCLIPPER_TEST_INT", "")
	if got := GetEnvInt("YTCLIPPER_TEST_INT", 7); got != 7 {
		t.Errorf("unset: got %d; want the fallback 7", got)
	}
	t.Setenv("YTCLIPPER_TEST_INT", "abc")
	if got := GetEnvInt("YTCLIPPER_TEST_INT", 7); got != 7 {
		t.Errorf("not a number: got %d; want the fallback 7", got)
	}
	t.Setenv("YTCLIPPER_TEST_INT", "42")
	if got := GetEnvInt("YTCLIPPER_TEST_INT", 7); got != 42 {
		t.Errorf("set: got %d; want 42", got)
	}
}

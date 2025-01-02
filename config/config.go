package config

import (
	"os"
	"strconv"
)

const (
	CONFIG_KEY_PORT                    			= "YTCLIPPER_PORT"
	CONFIG_KEY_DEBUG                    		= "YTCLIPPER_DEBUG"
	CONFIG_KEY_CLIP_SIZE_LIMIT_IN_MB         	= "YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB"
	CONFIG_KEY_YOUTUBE_COOKIES         			= "YOUTUBE_COOKIES"
	CONFIG_KEY_RATE_LIMITER_RATE       			= "YTCLIPPER_RATE_LIMITER_RATE"
	CONFIG_KEY_RATE_LIMITER_BURST      			= "YTCLIPPER_RATE_LIMITER_BURST"
	CONFIG_KEY_RATE_LIMITER_EXPIRES_IN_MINUTES	= "YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES"
)

var CONFIG *Config = NewConfig()

type Config struct {
	Port			string
	ClipSizeInMb 	int64
	Debug			bool
	YoutubeCookies	string
	RateLimiterConfig RateLimiterConfig
}

type RateLimiterConfig struct {
	Rate 	float64
	Burst	int
	ExpiresInMinutes int
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
	youtubeCookies := GetEnv(CONFIG_KEY_YOUTUBE_COOKIES, "")
	clipSizeInMb := GetClipSizeLimit()

	return &Config{
		Port:  port,
		Debug: debug,
		YoutubeCookies: youtubeCookies,
		ClipSizeInMb: clipSizeInMb,
		RateLimiterConfig:  *NewRateLimiterConfig(),
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


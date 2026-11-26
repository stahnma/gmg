// Package config loads server settings from GMG_* environment variables.
// Variable names and defaults match the original Node server.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config holds all server settings.
type Config struct {
	HTTPPort        string        // PORT
	GrillHost       string        // GMG_GRILL_HOST; a "255" address means discover by broadcast
	GrillPort       int           // GMG_GRILL_PORT
	GrillTries      int           // GMG_GRILL_TRIES
	GrillRetry      time.Duration // GMG_GRILL_RETRYMS
	PollingInterval time.Duration // GMG_STATUS_POLLING_INTERVAL (ms)
	DBPath          string        // GMG_DB_PATH: directory holding grill_data.db
	DBMaxSizeMB     int           // GMG_DB_MAX_SIZE_MB
	SlackWebhookURL string        // GMG_ALERTS_SLACK_WEBHOOKURL; Slack alerts are off when empty
	SlackChannel    string        // GMG_ALERTS_SLACK_CHANNEL
	SlackUsername   string        // GMG_ALERTS_SLACK_USERNAME
	AudioEnabled    bool          // GMG_ALERTS_AUDIO_ENABLED
	LogLevel        string        // GMG_LOG_LEVEL: debug, info, warn, error
}

// Default returns the defaults used when a variable is unset.
func Default() Config {
	return Config{
		HTTPPort:        "3001",
		GrillHost:       "255.255.255.255",
		GrillPort:       8080,
		GrillTries:      5,
		GrillRetry:      2000 * time.Millisecond,
		PollingInterval: 5000 * time.Millisecond,
		DBPath:          "data",
		DBMaxSizeMB:     50,
		SlackChannel:    "#alerts",
		SlackUsername:   "grill-bot",
		AudioEnabled:    false,
		LogLevel:        "info",
	}
}

// Load reads the configuration using getenv (usually os.Getenv).
func Load(getenv func(string) string) (Config, error) {
	c := Default()
	var errs []string
	str := func(name string, dst *string) {
		if v := strings.TrimSpace(getenv(name)); v != "" {
			*dst = v
		}
	}
	num := func(name string, dst *int) {
		v := strings.TrimSpace(getenv(name))
		if v == "" {
			return
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			errs = append(errs, fmt.Sprintf("%s must be a non-negative integer, got %q", name, v))
			return
		}
		*dst = n
	}
	millis := func(name string, dst *time.Duration) {
		n := int(*dst / time.Millisecond)
		num(name, &n)
		*dst = time.Duration(n) * time.Millisecond
	}

	str("PORT", &c.HTTPPort)
	str("GMG_GRILL_HOST", &c.GrillHost)
	num("GMG_GRILL_PORT", &c.GrillPort)
	num("GMG_GRILL_TRIES", &c.GrillTries)
	millis("GMG_GRILL_RETRYMS", &c.GrillRetry)
	millis("GMG_STATUS_POLLING_INTERVAL", &c.PollingInterval)
	str("GMG_DB_PATH", &c.DBPath)
	num("GMG_DB_MAX_SIZE_MB", &c.DBMaxSizeMB)
	str("GMG_ALERTS_SLACK_WEBHOOKURL", &c.SlackWebhookURL)
	str("GMG_ALERTS_SLACK_CHANNEL", &c.SlackChannel)
	str("GMG_ALERTS_SLACK_USERNAME", &c.SlackUsername)
	str("GMG_LOG_LEVEL", &c.LogLevel)
	if v := strings.TrimSpace(getenv("GMG_ALERTS_AUDIO_ENABLED")); v != "" {
		// Same rule as the Node server: only "true" enables audio.
		c.AudioEnabled = strings.EqualFold(v, "true")
	}

	if c.PollingInterval <= 0 {
		errs = append(errs, "GMG_STATUS_POLLING_INTERVAL must be greater than 0")
	}
	if c.GrillRetry <= 0 {
		errs = append(errs, "GMG_GRILL_RETRYMS must be greater than 0")
	}
	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}
	return c, nil
}

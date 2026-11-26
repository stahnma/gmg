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
	HTTPPort         string        // PORT
	GrillHost        string        // GMG_GRILL_HOST; a "255" address means discover by broadcast
	GrillPort        int           // GMG_GRILL_PORT
	GrillTries       int           // GMG_GRILL_TRIES
	GrillRetry       time.Duration // GMG_GRILL_RETRYMS
	PollingInterval  time.Duration // GMG_STATUS_POLLING_INTERVAL (ms)
	DBPath           string        // GMG_DB_PATH: directory holding grill_data.db
	DBMaxSizeMB      int           // GMG_DB_MAX_SIZE_MB
	SlackWebhookURL  string        // GMG_ALERTS_SLACK_WEBHOOKURL; Slack alerts are off when empty
	SlackChannel     string        // GMG_ALERTS_SLACK_CHANNEL
	SlackUsername    string        // GMG_ALERTS_SLACK_USERNAME
	AudioEnabled     bool          // GMG_ALERTS_AUDIO_ENABLED
	UnreachableAfter time.Duration // GMG_ALERTS_UNREACHABLE_SECONDS; 0 disables the alert
	LogLevel         string        // GMG_LOG_LEVEL: debug, info, warn, error
	TempDropDegrees  int           // GMG_ALERTS_TEMPDROP_DEGREES; 0 disables the alert
	TempDropFor      time.Duration // GMG_ALERTS_TEMPDROP_MINUTES
	IgnitionDegrees  int           // GMG_ALERTS_IGNITION_DEGREES; 0 disables the alert
	IgnitionWithin   time.Duration // GMG_ALERTS_IGNITION_MINUTES
	GrillTempMin     int           // GMG_GRILL_TEMP_MIN: lowest grill target accepted (°F)
	GrillTempMax     int           // GMG_GRILL_TEMP_MAX
	FoodTempMin      int           // GMG_FOOD_TEMP_MIN: lowest food target accepted (°F)
	FoodTempMax      int           // GMG_FOOD_TEMP_MAX
	CORSOrigins      string        // GMG_CORS_ORIGINS: comma-separated origins allowed to call the API; "*" for any
}

// Default returns the defaults used when a variable is unset.
func Default() Config {
	return Config{
		HTTPPort:         "3001",
		GrillHost:        "255.255.255.255",
		GrillPort:        8080,
		GrillTries:       5,
		GrillRetry:       2000 * time.Millisecond,
		PollingInterval:  5000 * time.Millisecond,
		DBPath:           "data",
		DBMaxSizeMB:      50,
		SlackChannel:     "#alerts",
		SlackUsername:    "grill-bot",
		AudioEnabled:     false,
		UnreachableAfter: 60 * time.Second,
		TempDropDegrees:  50,
		TempDropFor:      5 * time.Minute,
		IgnitionDegrees:  30,
		IgnitionWithin:   15 * time.Minute,
		GrillTempMin:     150,
		GrillTempMax:     550,
		FoodTempMin:      0,
		FoodTempMax:      300,
		LogLevel:         "info",
	}
}

// Origins returns CORSOrigins as a list, without blanks or trailing slashes.
func (c Config) Origins() []string {
	var out []string
	for _, o := range strings.Split(c.CORSOrigins, ",") {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			out = append(out, o)
		}
	}
	return out
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
	seconds := func(name string, dst *time.Duration) {
		n := int(*dst / time.Second)
		num(name, &n)
		*dst = time.Duration(n) * time.Second
	}
	minutes := func(name string, dst *time.Duration) {
		n := int(*dst / time.Minute)
		num(name, &n)
		*dst = time.Duration(n) * time.Minute
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
	num("GMG_GRILL_TEMP_MIN", &c.GrillTempMin)
	num("GMG_GRILL_TEMP_MAX", &c.GrillTempMax)
	num("GMG_FOOD_TEMP_MIN", &c.FoodTempMin)
	num("GMG_FOOD_TEMP_MAX", &c.FoodTempMax)
	if c.GrillTempMin > c.GrillTempMax {
		errs = append(errs, "GMG_GRILL_TEMP_MIN must not be above GMG_GRILL_TEMP_MAX")
	}
	if c.FoodTempMin > c.FoodTempMax {
		errs = append(errs, "GMG_FOOD_TEMP_MIN must not be above GMG_FOOD_TEMP_MAX")
	}
	if c.GrillTempMax > 999 || c.FoodTempMax > 999 {
		errs = append(errs, "temperature limits must be at most 999 °F (the protocol sends 3 digits)")
	}
	str("GMG_CORS_ORIGINS", &c.CORSOrigins)
	seconds("GMG_ALERTS_UNREACHABLE_SECONDS", &c.UnreachableAfter)
	num("GMG_ALERTS_TEMPDROP_DEGREES", &c.TempDropDegrees)
	minutes("GMG_ALERTS_TEMPDROP_MINUTES", &c.TempDropFor)
	num("GMG_ALERTS_IGNITION_DEGREES", &c.IgnitionDegrees)
	minutes("GMG_ALERTS_IGNITION_MINUTES", &c.IgnitionWithin)
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

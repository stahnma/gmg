package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDefaults(t *testing.T) {
	c, err := Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c != Default() {
		t.Fatalf("got %+v", c)
	}
	if c.SlackWebhookURL != "" || c.AudioEnabled || c.GrillHost != "255.255.255.255" || c.PollingInterval != 5*time.Second {
		t.Fatalf("unexpected defaults %+v", c)
	}
}

func TestFromEnvironment(t *testing.T) {
	c, err := Load(env(map[string]string{
		"PORT": "80", "GMG_GRILL_HOST": "192.168.1.50", "GMG_GRILL_PORT": "9000", "GMG_GRILL_TRIES": "3",
		"GMG_GRILL_RETRYMS": "1500", "GMG_STATUS_POLLING_INTERVAL": "2000", "GMG_DB_PATH": "/data",
		"GMG_DB_MAX_SIZE_MB": "10", "GMG_ALERTS_SLACK_WEBHOOKURL": "https://hooks.slack.com/services/x",
		"GMG_ALERTS_UNREACHABLE_SECONDS": "90",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.HTTPPort, want.GrillHost, want.GrillPort, want.GrillTries = "80", "192.168.1.50", 9000, 3
	want.GrillRetry, want.PollingInterval = 1500*time.Millisecond, 2*time.Second
	want.DBPath, want.DBMaxSizeMB, want.SlackWebhookURL = "/data", 10, "https://hooks.slack.com/services/x"
	want.UnreachableAfter = 90 * time.Second
	if c != want {
		t.Fatalf("got  %+v\nwant %+v", c, want)
	}
}

func TestAudioFlag(t *testing.T) {
	for v, want := range map[string]bool{"": false, "true": true, "TRUE": true, "false": false, "0": false, "no": false} {
		c, err := Load(env(map[string]string{"GMG_ALERTS_AUDIO_ENABLED": v}))
		if err != nil || c.AudioEnabled != want {
			t.Errorf("GMG_ALERTS_AUDIO_ENABLED=%q: got %v (%v), want %v", v, c.AudioEnabled, err, want)
		}
	}
}

func TestInvalidNumbers(t *testing.T) {
	_, err := Load(env(map[string]string{"GMG_GRILL_PORT": "abc", "GMG_STATUS_POLLING_INTERVAL": "0"}))
	if err == nil || !strings.Contains(err.Error(), "GMG_GRILL_PORT") || !strings.Contains(err.Error(), "GMG_STATUS_POLLING_INTERVAL") {
		t.Fatalf("expected errors naming both variables, got %v", err)
	}
}

func TestOrigins(t *testing.T) {
	c, _ := Load(env(map[string]string{"GMG_CORS_ORIGINS": " https://a.example/, ,http://b.example:8080 "}))
	if got := c.Origins(); len(got) != 2 || got[0] != "https://a.example" || got[1] != "http://b.example:8080" {
		t.Fatalf("got %q", got)
	}
	if got := Default().Origins(); len(got) != 0 {
		t.Fatalf("default: %q", got)
	}
}

func TestTemperatureLimits(t *testing.T) {
	c, err := Load(env(map[string]string{"GMG_GRILL_TEMP_MIN": "180", "GMG_GRILL_TEMP_MAX": "500", "GMG_FOOD_TEMP_MAX": "250"}))
	if err != nil || c.GrillTempMin != 180 || c.GrillTempMax != 500 || c.FoodTempMin != 0 || c.FoodTempMax != 250 {
		t.Fatalf("got %+v %v", c, err)
	}
	for _, m := range []map[string]string{
		{"GMG_GRILL_TEMP_MIN": "600"},
		{"GMG_FOOD_TEMP_MIN": "400"},
		{"GMG_GRILL_TEMP_MAX": "1000"},
	} {
		if _, err := Load(env(m)); err == nil {
			t.Errorf("%v: expected an error", m)
		}
	}
}

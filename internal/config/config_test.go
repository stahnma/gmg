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
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.HTTPPort, want.GrillHost, want.GrillPort, want.GrillTries = "80", "192.168.1.50", 9000, 3
	want.GrillRetry, want.PollingInterval = 1500*time.Millisecond, 2*time.Second
	want.DBPath, want.DBMaxSizeMB, want.SlackWebhookURL = "/data", 10, "https://hooks.slack.com/services/x"
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

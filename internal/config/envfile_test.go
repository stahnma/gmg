package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func lookup(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnvFile(t *testing.T) {
	p := writeEnvFile(t, `# grill settings
GMG_GRILL_HOST=192.168.1.50
export GMG_STATUS_POLLING_INTERVAL=2000
PORT="8000"
GMG_ALERTS_SLACK_WEBHOOKURL='https://hooks.slack.com/services/x' # comment
GMG_LOG_LEVEL=debug
`)
	// The environment overrides the file, including variables set to "".
	getenv, loaded, err := WithEnvFile(lookup(map[string]string{"PORT": "9000", "GMG_LOG_LEVEL": ""}), p, true)
	if err != nil || !loaded {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}
	c, err := Load(getenv)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.GrillHost, want.PollingInterval, want.HTTPPort = "192.168.1.50", 2*time.Second, "9000"
	want.SlackWebhookURL = "https://hooks.slack.com/services/x"
	if c != want {
		t.Fatalf("got  %+v\nwant %+v", c, want)
	}
}

func TestEnvFileMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), ".env")

	getenv, loaded, err := WithEnvFile(lookup(map[string]string{"PORT": "80"}), missing, false)
	if err != nil || loaded || getenv("PORT") != "80" {
		t.Fatalf("optional missing file: loaded=%v err=%v PORT=%q", loaded, err, getenv("PORT"))
	}
	if _, _, err := WithEnvFile(lookup(nil), missing, true); err == nil {
		t.Fatal("required missing file: expected an error")
	}
}

func TestEnvFileUnreadable(t *testing.T) {
	// A directory where the file should be is an error even when optional.
	if _, _, err := WithEnvFile(lookup(nil), t.TempDir(), false); err == nil {
		t.Fatal("expected an error")
	}
}

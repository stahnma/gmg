package protocol

import (
	"encoding/hex"
	"testing"
)

// buf builds a status buffer the way the original JS tests did (32 bytes,
// fields at their documented offsets).
func buf(grill, food, setGrill, lowPellet, setFood int, state byte) []byte {
	b := make([]byte, 32)
	put := func(off, v int) { b[off], b[off+1] = byte(v%256), byte(v/256) }
	put(2, grill)
	put(4, food)
	put(6, setGrill)
	put(24, lowPellet)
	put(28, setFood)
	b[30] = state
	return b
}

func mustParse(t *testing.T, b []byte) Status {
	t.Helper()
	s, err := ParseStatus(b)
	if err != nil {
		t.Fatalf("ParseStatus: %v", err)
	}
	return s
}

func TestParseStatusState(t *testing.T) {
	tests := []struct {
		state     byte
		want      string
		isOn, fan bool
	}{
		{0, "off", false, false},
		{1, "on", true, false},
		{2, "fan mode", false, true},
		{5, "unknown", false, false},
		{0x11, "on", true, false}, // only the low nibble is the state
	}
	for _, tt := range tests {
		s := mustParse(t, buf(0, 0, 0, 0, 0, tt.state))
		if s.State != tt.want || s.IsOn != tt.isOn || s.FanModeActive != tt.fan {
			t.Errorf("state %#x: got %q isOn=%v fan=%v, want %q isOn=%v fan=%v",
				tt.state, s.State, s.IsOn, s.FanModeActive, tt.want, tt.isOn, tt.fan)
		}
	}
}

func TestParseStatusTemperatures(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		get  func(Status) int
		want int
	}{
		{"current grill", buf(225, 0, 0, 0, 0, 1), func(s Status) int { return s.CurrentGrillTemp }, 225},
		{"current grill high byte", buf(500, 0, 0, 0, 0, 1), func(s Status) int { return s.CurrentGrillTemp }, 500},
		{"desired grill when on", buf(0, 0, 350, 0, 0, 1), func(s Status) int { return s.DesiredGrillTemp }, 350},
		{"desired grill when off", buf(0, 0, 350, 0, 0, 0), func(s Status) int { return s.DesiredGrillTemp }, 0},
		{"current food", buf(0, 165, 0, 0, 0, 1), func(s Status) int { return s.CurrentFoodTemp }, 165},
		{"food probe missing (557)", buf(0, 557, 0, 0, 0, 1), func(s Status) int { return s.CurrentFoodTemp }, 0},
		{"food probe missing (600)", buf(0, 600, 0, 0, 0, 1), func(s Status) int { return s.CurrentFoodTemp }, 0},
		{"food just below missing", buf(0, 556, 0, 0, 0, 1), func(s Status) int { return s.CurrentFoodTemp }, 556},
		{"desired food when on", buf(0, 0, 0, 0, 200, 1), func(s Status) int { return s.DesiredFoodTemp }, 200},
		{"desired food when off", buf(0, 0, 0, 0, 200, 0), func(s Status) int { return s.DesiredFoodTemp }, 0},
	}
	for _, tt := range tests {
		if got := tt.get(mustParse(t, tt.b)); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestParseStatusLowPellet(t *testing.T) {
	for v, want := range map[int]bool{128: true, 0: false, 64: false} {
		if got := mustParse(t, buf(0, 0, 0, v, 0, 1)).LowPelletAlarmActive; got != want {
			t.Errorf("warn %d: got %v, want %v", v, got, want)
		}
	}
}

func TestParseStatusTooShort(t *testing.T) {
	if _, err := ParseStatus([]byte("OK")); err == nil {
		t.Fatal("expected an error for a 2-byte response")
	}
}

// Captured from the original C# emulator: grill 90°F, probe 80°F, grill on.
const csharpEmulatorStatus = "55525a0050000000000000000000000000000000ffffffff000000000000010000000000"

func TestEncodeMatchesCSharpEmulator(t *testing.T) {
	got := hex.EncodeToString(RawStatus{GrillTemp: 90, ProbeTemp: 80, State: StateOn}.Encode())
	if got != csharpEmulatorStatus {
		t.Fatalf("encoded status differs from C# emulator\n got %s\nwant %s", got, csharpEmulatorStatus)
	}
}

func TestEncodeParseRoundTrip(t *testing.T) {
	raw := RawStatus{GrillTemp: 412, ProbeTemp: 163, GrillSetTemp: 450, ProbeSetTemp: 165, State: StateOn, Warn: WarnLowPellet}
	s := mustParse(t, raw.Encode())
	want := Status{State: "on", Hex: hex.EncodeToString(raw.Encode()), IsOn: true, CurrentGrillTemp: 412,
		DesiredGrillTemp: 450, CurrentFoodTemp: 163, DesiredFoodTemp: 165, LowPelletAlarmActive: true}
	if s != want {
		t.Fatalf("round trip:\n got %+v\nwant %+v", s, want)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		in   string
		want Command
	}{
		{"UR001!", Command{Kind: KindGetStatus}},
		{"UK001!", Command{Kind: KindPowerOn}},
		{"UK004!", Command{Kind: KindPowerOff}},
		{"UL!", Command{Kind: KindGetGrillID}},
		{"UN!", Command{Kind: KindInit}},
		{SetGrillTemp(225), Command{Kind: KindSetGrillTemp, Temp: 225}},
		{SetFoodTemp(165), Command{Kind: KindSetFoodTemp, Temp: 165}},
		{SetFoodTemp(50), Command{Kind: KindSetFoodTemp, Temp: 50}}, // C# emulator rejected 2 digits
		{"UT1000!", Command{Kind: KindUnknown}},
		{"UT9!", Command{Kind: KindUnknown}},
		{"UA!", Command{Kind: KindUnknown}},
		{"", Command{Kind: KindUnknown}},
	}
	for _, tt := range tests {
		if got := ParseCommand(tt.in); got != tt.want {
			t.Errorf("ParseCommand(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

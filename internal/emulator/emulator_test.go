package emulator

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

func status(t *testing.T, g *Grill) protocol.Status {
	t.Helper()
	s, err := protocol.ParseStatus(g.Handle(protocol.Command{Kind: protocol.KindGetStatus}))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func do(g *Grill, kind protocol.CommandKind, temp ...int) string {
	c := protocol.Command{Kind: kind}
	if len(temp) > 0 {
		c.Temp = temp[0]
	}
	return string(g.Handle(c))
}

func TestStartsOffAtAmbient(t *testing.T) {
	g := New()
	s := status(t, g)
	if s.IsOn || s.CurrentGrillTemp != 90 || s.CurrentFoodTemp != 80 {
		t.Fatalf("got %+v", s)
	}
	g.Tick() // ramp period already elapsed: snaps to clamped target (0 -> ambient)
	if s := status(t, g); s.CurrentGrillTemp != 90 || s.CurrentFoodTemp != 80 {
		t.Fatalf("after tick: %+v", s)
	}
}

func TestPowerAndTargets(t *testing.T) {
	g := New()
	if r := do(g, protocol.KindPowerOn); r != "OK" {
		t.Fatalf("power on: %q", r)
	}
	do(g, protocol.KindSetGrillTemp, 225)
	do(g, protocol.KindSetFoodTemp, 165)
	s := status(t, g)
	if !s.IsOn || s.DesiredGrillTemp != 225 || s.DesiredFoodTemp != 165 {
		t.Fatalf("got %+v", s)
	}
	do(g, protocol.KindPowerOff)
	s = status(t, g)
	if s.IsOn || s.DesiredGrillTemp != 0 || s.DesiredFoodTemp != 0 {
		t.Fatalf("after power off: %+v", s)
	}
}

func TestRampRates(t *testing.T) {
	g := New()
	do(g, protocol.KindPowerOn)
	do(g, protocol.KindSetGrillTemp, 110)
	for i := 0; i < 5; i++ {
		g.Tick()
	}
	if s := status(t, g); s.CurrentGrillTemp != 100 || s.CurrentFoodTemp != 85 {
		t.Fatalf("after 5s want grill 100 / probe 85, got %d / %d", s.CurrentGrillTemp, s.CurrentFoodTemp)
	}
	for i := 0; i < 20; i++ {
		g.Tick()
	}
	if s := status(t, g); s.CurrentGrillTemp != 110 {
		t.Fatalf("grill should stop at target 110, got %d", s.CurrentGrillTemp)
	}
}

func TestClampsAndSnap(t *testing.T) {
	g := New()
	do(g, protocol.KindPowerOn)
	do(g, protocol.KindSetGrillTemp, 999)
	for i := 0; i < rampSeconds+1; i++ {
		g.Tick()
	}
	if s := status(t, g); s.CurrentGrillTemp != 550 || s.CurrentFoodTemp != 550 {
		t.Fatalf("want clamp at 550/550 after ramp, got %d/%d", s.CurrentGrillTemp, s.CurrentFoodTemp)
	}
	do(g, protocol.KindPowerOff) // target 0 -> cools back to ambient
	for i := 0; i < rampSeconds+1; i++ {
		g.Tick()
	}
	if s := status(t, g); s.CurrentGrillTemp != 90 || s.CurrentFoodTemp != 80 {
		t.Fatalf("want ambient 90/80 after cooling, got %d/%d", s.CurrentGrillTemp, s.CurrentFoodTemp)
	}
}

func TestLowPelletEveryThirdPowerOn(t *testing.T) {
	g := New()
	var got []bool
	for i := 0; i < 6; i++ {
		do(g, protocol.KindPowerOn)
		do(g, protocol.KindPowerOn) // repeated power-on while on doesn't count
		got = append(got, status(t, g).LowPelletAlarmActive)
		do(g, protocol.KindPowerOff)
	}
	want := []bool{false, false, true, false, false, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("low pellet by power-on: got %v, want %v", got, want)
		}
	}
}

func TestIDAndInit(t *testing.T) {
	g := New()
	if r := do(g, protocol.KindGetGrillID); r != GrillID {
		t.Fatalf("grill id: %q", r)
	}
	if r := do(g, protocol.KindInit); len(r) != 36 {
		t.Fatalf("init should return a firmware id, got %q", r)
	}
}

func TestServeOverUDP(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, conn, New(), slog.New(slog.NewTextHandler(io.Discard, nil))) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve: %v", err)
		}
	}()

	client, err := net.Dial("udp4", conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	roundTrip := func(req string) ([]byte, error) {
		if _, err := client.Write([]byte(req)); err != nil {
			return nil, err
		}
		client.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		b := make([]byte, 1024)
		n, err := client.Read(b)
		return b[:n], err
	}

	if r, err := roundTrip("UK001!"); err != nil || string(r) != "OK" {
		t.Fatalf("power on: %q %v", r, err)
	}
	r, err := roundTrip("UR001!")
	if err != nil || len(r) != protocol.StatusSize {
		t.Fatalf("status: %d bytes, %v", len(r), err)
	}
	if s, _ := protocol.ParseStatus(r); !s.IsOn {
		t.Fatalf("status after power on: %+v", s)
	}
	if _, err := roundTrip("UA!"); err == nil {
		t.Fatal("unsupported command should get no response")
	}
}

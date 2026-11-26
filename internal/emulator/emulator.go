// Package emulator simulates a Green Mountain Grill on the UDP protocol, for
// development and testing without a real grill.
package emulator

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/stahnma/gmg/internal/protocol"
)

const (
	maxGrillTemp = 550
	minGrillTemp = 90  // ambient reading for the grill
	minProbeTemp = 80  // ambient reading for the food probe
	rampSeconds  = 500 // after a change, ramp for this many ticks, then snap to target
	// GrillID is what the emulator reports for the grill ID command.
	GrillID = "GMG10116294"
)

// Grill is the emulated grill state. Tick advances the simulation by one
// second; Handle processes a command. Safe for concurrent use.
type Grill struct {
	mu               sync.Mutex
	on               bool
	targetGrill      int
	targetProbe      int
	currentGrill     int
	currentProbe     int
	ticksSinceChange int
	powerOns         int
}

// New returns a grill that is off and at ambient temperature.
func New() *Grill {
	return &Grill{currentGrill: minGrillTemp, currentProbe: minProbeTemp, ticksSinceChange: rampSeconds}
}

// Handle applies cmd and returns the response bytes, or nil for commands the
// grill doesn't answer.
func (g *Grill) Handle(cmd protocol.Command) []byte {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch cmd.Kind {
	case protocol.KindGetStatus:
		return g.status().Encode()
	case protocol.KindPowerOn:
		if !g.on {
			g.powerOns++
		}
		g.on = true
		return []byte(protocol.ResponseOK)
	case protocol.KindPowerOff:
		g.on = false
		g.targetGrill, g.targetProbe = 0, 0
		g.ticksSinceChange = 0
		return []byte(protocol.ResponseOK)
	case protocol.KindSetGrillTemp:
		g.targetGrill = cmd.Temp
		g.ticksSinceChange = 0
		return []byte(protocol.ResponseOK)
	case protocol.KindSetFoodTemp:
		g.targetProbe = cmd.Temp
		return []byte(protocol.ResponseOK)
	case protocol.KindGetGrillID:
		return []byte(GrillID)
	case protocol.KindInit:
		return []byte(randomFirmwareID())
	}
	return nil
}

func (g *Grill) status() protocol.RawStatus {
	s := protocol.RawStatus{
		GrillTemp:    uint16(g.currentGrill),
		ProbeTemp:    uint16(g.currentProbe),
		GrillSetTemp: uint16(g.targetGrill),
		ProbeSetTemp: uint16(g.targetProbe),
		State:        protocol.StateOff,
	}
	if g.on {
		s.State = protocol.StateOn
	}
	// Every third power-on reports a low pellet warning, to exercise alerts.
	if g.powerOns > 0 && g.powerOns%3 == 0 {
		s.Warn = protocol.WarnLowPellet
	}
	return s
}

// Tick advances the simulation by one second: temperatures move toward the
// grill target (grill 2°F/s, probe 1°F/s) and snap to it once the ramp
// period has passed. Mirrors the original C# emulator, including the probe
// following the grill target rather than the probe target.
func (g *Grill) Tick() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ticksSinceChange < rampSeconds {
		g.ticksSinceChange++
		if g.targetGrill > g.currentGrill {
			g.currentGrill = clamp(min(g.currentGrill+2, g.targetGrill), minGrillTemp, maxGrillTemp)
		} else {
			g.currentGrill = clamp(max(g.currentGrill-2, g.targetGrill), minGrillTemp, maxGrillTemp)
		}
		if g.targetGrill > g.currentGrill {
			g.currentProbe = clamp(min(g.currentProbe+1, g.targetGrill), minProbeTemp, maxGrillTemp)
		} else {
			g.currentProbe = clamp(max(g.currentProbe-1, g.targetGrill), minProbeTemp, maxGrillTemp)
		}
		return
	}
	g.currentGrill = clamp(g.targetGrill, minGrillTemp, maxGrillTemp)
	g.currentProbe = clamp(g.targetGrill, minProbeTemp, maxGrillTemp)
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

func randomFirmwareID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

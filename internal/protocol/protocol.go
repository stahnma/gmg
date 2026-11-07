// Package protocol implements the Green Mountain Grill WiFi UDP protocol:
// the ASCII command strings and the 36-byte binary status ("UR") response.
// It is shared by the grill client and the emulator so both agree on the
// wire format.
package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
)

// Commands sent to the grill. Each is an ASCII string terminated by '!'.
const (
	CmdGetStatus   = "UR001!"
	CmdPowerOn     = "UK001!"
	CmdPowerOff    = "UK004!"
	CmdGetGrillID  = "UL!"
	CmdInit        = "UN!" // also returns firmware info
	ResponseOK     = "OK"
	StatusSize     = 36
	minStatusBytes = 31 // the state byte is at offset 30
)

// SetGrillTemp returns the command to set the grill target temperature (°F).
func SetGrillTemp(f int) string { return fmt.Sprintf("UT%d!", f) }

// SetFoodTemp returns the command to set the food probe target temperature (°F).
func SetFoodTemp(f int) string { return fmt.Sprintf("UF%d!", f) }

// Byte offsets within the status response.
const (
	offGrillTemp     = 2  // uint16 LE
	offProbeTemp     = 4  // uint16 LE
	offGrillSetTemp  = 6  // uint16 LE
	offCurveRemain   = 20 // 4 bytes, 0xFF when unused
	offWarnCode      = 24 // uint16 LE
	offProbeSetTemp  = 28 // uint16 LE
	offGrillState    = 30 // low nibble: 0 off, 1 on, 2 fan mode
	foodProbeMissing = 557
)

// Grill power states as reported in the status response.
const (
	StateOff byte = 0
	StateOn  byte = 1
	StateFan byte = 2
)

// Warn codes as reported in the status response.
const (
	WarnNone      uint16 = 0
	WarnLowPellet uint16 = 128
)

// Status is the decoded grill status. JSON field names and order match the
// original Node server's API so existing clients keep working.
type Status struct {
	State                string `json:"state"`
	Hex                  string `json:"_hex"`
	IsOn                 bool   `json:"isOn"`
	CurrentGrillTemp     int    `json:"currentGrillTemp"`
	DesiredGrillTemp     int    `json:"desiredGrillTemp"`
	CurrentFoodTemp      int    `json:"currentFoodTemp"`
	DesiredFoodTemp      int    `json:"desiredFoodTemp"`
	FanModeActive        bool   `json:"fanModeActive"`
	LowPelletAlarmActive bool   `json:"lowPelletAlarmActive"`
}

// ParseStatus decodes a status response from the grill.
func ParseStatus(b []byte) (Status, error) {
	if len(b) < minStatusBytes {
		return Status{}, fmt.Errorf("status response too short: %d bytes", len(b))
	}
	u16 := func(off int) int { return int(binary.LittleEndian.Uint16(b[off:])) }

	s := Status{Hex: hex.EncodeToString(b)}
	switch b[offGrillState] & 0x0F {
	case StateOff:
		s.State = "off"
	case StateOn:
		s.State = "on"
	case StateFan:
		s.State = "fan mode"
	default:
		s.State = "unknown"
	}
	s.IsOn = s.State == "on"
	s.FanModeActive = s.State == "fan mode"
	s.CurrentGrillTemp = u16(offGrillTemp)
	s.CurrentFoodTemp = u16(offProbeTemp)
	if s.CurrentFoodTemp >= foodProbeMissing { // no probe connected
		s.CurrentFoodTemp = 0
	}
	if s.IsOn { // targets are meaningless while the grill is off
		s.DesiredGrillTemp = u16(offGrillSetTemp)
		s.DesiredFoodTemp = u16(offProbeSetTemp)
	}
	s.LowPelletAlarmActive = uint16(u16(offWarnCode)) == WarnLowPellet
	return s, nil
}

// RawStatus holds the fields the emulator encodes into a status response.
type RawStatus struct {
	GrillTemp, ProbeTemp, GrillSetTemp, ProbeSetTemp uint16
	State                                            byte
	Warn                                             uint16
}

// Encode returns the 36-byte status response for s.
func (s RawStatus) Encode() []byte {
	b := make([]byte, StatusSize)
	copy(b, "UR")
	binary.LittleEndian.PutUint16(b[offGrillTemp:], s.GrillTemp)
	binary.LittleEndian.PutUint16(b[offProbeTemp:], s.ProbeTemp)
	binary.LittleEndian.PutUint16(b[offGrillSetTemp:], s.GrillSetTemp)
	for i := 0; i < 4; i++ {
		b[offCurveRemain+i] = 0xFF
	}
	binary.LittleEndian.PutUint16(b[offWarnCode:], s.Warn)
	binary.LittleEndian.PutUint16(b[offProbeSetTemp:], s.ProbeSetTemp)
	b[offGrillState] = s.State
	return b
}

// CommandKind identifies a parsed command.
type CommandKind int

const (
	KindUnknown CommandKind = iota
	KindGetStatus
	KindPowerOn
	KindPowerOff
	KindGetGrillID
	KindInit
	KindSetGrillTemp
	KindSetFoodTemp
)

// Command is a parsed request, as received by the emulator.
type Command struct {
	Kind CommandKind
	Temp int // for KindSetGrillTemp / KindSetFoodTemp
}

var setTempRe = regexp.MustCompile(`^U([TF])(\d{2,3})!$`)

// ParseCommand parses an ASCII command. It returns KindUnknown for anything
// it doesn't recognise.
func ParseCommand(s string) Command {
	switch s {
	case CmdGetStatus:
		return Command{Kind: KindGetStatus}
	case CmdPowerOn:
		return Command{Kind: KindPowerOn}
	case CmdPowerOff:
		return Command{Kind: KindPowerOff}
	case CmdGetGrillID:
		return Command{Kind: KindGetGrillID}
	case CmdInit:
		return Command{Kind: KindInit}
	}
	if m := setTempRe.FindStringSubmatch(s); m != nil {
		t, _ := strconv.Atoi(m[2])
		if m[1] == "T" {
			return Command{Kind: KindSetGrillTemp, Temp: t}
		}
		return Command{Kind: KindSetFoodTemp, Temp: t}
	}
	return Command{Kind: KindUnknown}
}

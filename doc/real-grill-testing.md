# Testing on a real grill

Steps for checking the Go server against real hardware ([#16](https://github.com/stahnma/gmg/issues/16)). So far the server has only been run against the emulator and the captures in [`decode.txt`](decode.txt). Write down the results in the issue, along with the grill model and controller firmware.

Never leave the grill alone during these tests. Some of them, like pulling the probe or unplugging the controller, interrupt a cook on purpose.

## Setup

Run the server on the same network as the grill, with debug logging. Shorter alert windows let the alert tests finish in minutes:

```sh
make build
GMG_LOG_LEVEL=debug ./bin/gmg            # discovery: leave GMG_GRILL_HOST unset
GMG_GRILL_HOST=192.168.1.50 ./bin/gmg    # or with the grill's IP
```

In a second terminal, record every status frame. Use the same layout as `decode.txt` so new captures can be added there:

```sh
while true; do
  s=$(curl -s http://localhost:3001/api/status)
  printf '%s  %-8s %s\n' "$(date +%T)" "$(jq -r .state <<<"$s")" "$(jq -r ._hex <<<"$s" | sed -E 's/(.{4})/\1 /g')"
  sleep 5
done | tee -a captures.txt
```

`curl -N http://localhost:3001/api/events` shows the live `status`, `connection` and `alert` events.

## Checklist

### Connection
- [ ] **Discovery.** With `GMG_GRILL_HOST` unset, the log shows `grill discovered` with the grill's IP and ID.
- [ ] **Explicit IP.** With `GMG_GRILL_HOST=<ip>`, status arrives without discovery.
- [ ] **Rediscovery.** In discovery mode, give the grill a new DHCP lease (a DHCP reservation change and a reboot of the controller usually does it). After 3 failed polls the log shows `grill address changed`, and status resumes without restarting the server.

### Controls
- [ ] **Power.** Use Power on, off and toggle from the dashboard. Try power on while in fan mode: it should be refused with "Cannot start grill when fan mode is active."
- [ ] **Targets.** Set grill targets of 150 and 550 (both accepted) and 149 and 551 (both refused with a range message). Set food targets. If your model's range differs, note it and set `GMG_GRILL_TEMP_MIN`/`_MAX` to match.
- [ ] **Slow replies.** Commands that the grill acknowledges slowly (watch `attempt=` in the debug log) still succeed.

### Readings
- [ ] **Probe.** With the probe unplugged, the food card shows "No probe". Plug it in and a reading appears.
- [ ] **Fan mode.** After power off, `state` is `fan mode` until the grill cools, then `off`.
- [ ] **Accuracy.** Grill and probe temperatures match the controller's display.

### Alerts
Run these with the windows shortened, e.g.:

```sh
GMG_ALERTS_UNREACHABLE_SECONDS=30 GMG_ALERTS_TEMPDROP_DEGREES=10 GMG_ALERTS_TEMPDROP_MINUTES=1 \
GMG_ALERTS_IGNITION_DEGREES=200 GMG_ALERTS_IGNITION_MINUTES=2 ./bin/gmg
```

- [ ] **Target reached** for grill and food. Each fires once, and again after a new target or a power cycle.
- [ ] **Grill May Not Have Lit.** Turn the grill on from cold and set the grill target to 450 °F straight away. With the settings above, a normal start can't gain 200 °F in 2 minutes, so the alert should fire. This tests the alert without a failed igniter. (A target within 200 °F of the starting temperature holds the alert off, as it would for a hot restart.)
- [ ] **Grill Temperature Dropping.** Once the grill holds its target, open the lid until it falls more than 10 °F below the target for a minute.
- [ ] **Food Probe Disconnected.** With a food target set, pull the probe out of the controller.
- [ ] **Grill Not Responding / Responding Again.** While the grill is on, unplug its controller or block it on your Wi-Fi for longer than 30 seconds, then restore it. The dashboard should dim the readings and show "Grill not responding" while it's gone.
- [ ] **Low pellets.** If you can safely let the hopper run low, check that the alarm fires, and that it fires again after a refill and a second low.
- [ ] **Slack**, if used. Every alert above also reaches the channel.

### A full cook
- [ ] Run a multi-hour cook with the dashboard open on a phone. Afterwards, check that the history chart covers the whole cook and that no unexpected alerts fired.

## Captures to add to `decode.txt`

Label each frame the way the existing ones are labelled, and add any that are new:

- Startup and ignition, from pressing power until the target is reached
- Steady at target
- Cooling down in fan mode, then off
- Probe unplugged, and plugged in again, while on
- Low pellet alarm
- Any error the controller shows on its display (note the code)

Each new state can then become a golden test in `internal/protocol/protocol_test.go` (see `TestParseRealGrillCaptures`).

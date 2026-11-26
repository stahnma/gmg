# Green Mountain Grills App

A web dashboard for monitoring and controlling Green Mountain Grills over their WiFi (UDP) protocol, with reliable alerts for events like low pellets or a target temperature being reached.

This is a Go + Svelte rewrite of [Aenima4six2/gmg](https://github.com/Aenima4six2/gmg). It keeps the same HTTP API, configuration variables and history database, so it's a drop-in replacement. The last JavaScript/C# version is on the `integration` branch.

## Features
- Grill controls: power, grill and food target temperatures
- Live status and a temperature history chart
- Alerts in the browser (with optional sounds) and to Slack
- Timers: stopwatch and countdown
- Auto-discovery of the grill on the local network
- Grill emulator for development and testing without a grill
- REST API and a live event stream for extensions

## Preview
![preview](./assets/preview.png)

## Running the binary

The server is a single static binary with the web UI built in. Nothing else is needed at run time.

```sh
make build                   # bin/gmg and bin/gmg-emulator (needs Go, Node and Make: `flox activate`)
cp .env.example .env         # set GMG_GRILL_HOST, or leave it unset to discover the grill
./bin/gmg                    # http://localhost:3001
```

`gmg` reads `.env` from the working directory if one is there; use `--env-file /path/to/file` to read another file (it must exist). Environment variables override values in the file. History is kept in `./data` unless `GMG_DB_PATH` says otherwise.

To build for another machine, for example a Raspberry Pi: `make web && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o gmg ./cmd/gmg`.

## Running with Docker

```sh
docker build -t gmg .
docker run -d -p 80:80 -v gmg-data:/data -e GMG_GRILL_HOST=192.168.1.100 gmg
```

The image is a single static binary on distroless (about 17 MB) and runs as a non-root user. History is kept in the `/data` volume.

To let the server **discover the grill automatically**, leave `GMG_GRILL_HOST` unset and use host networking so the broadcast reaches your LAN:

```sh
docker run -d --network host -v gmg-data:/data gmg
```

Alternatively, copy `.env.example` to `.env`, set `GMG_GRILL_HOST`, and run `./start`.

### Trying it with the emulator

```sh
docker compose up --build    # or: make up
```

This starts the app and the grill emulator. The app is at http://localhost:3000.

## Configuration

Settings are environment variables, or the same names in a `.env` file (see [`.env.example`](.env.example)); environment variables take precedence.

| Variable | Description | Default |
|----------|-------------|---------|
| `GMG_GRILL_HOST` | Grill IP address. A broadcast address (containing `255`) means discover the grill on the network, and discover it again if it stops answering (for example after getting a new DHCP address) | `255.255.255.255` |
| `GMG_GRILL_PORT` | Grill UDP port | `8080` |
| `GMG_GRILL_TRIES` | Resends per command before giving up | `5` |
| `GMG_GRILL_RETRYMS` | Wait for a reply before resending (ms) | `2000` |
| `GMG_STATUS_POLLING_INTERVAL` | Status poll interval (ms) | `5000` |
| `GMG_DB_PATH` | Directory for the history database (`grill_data.db`) | `data` (`/data` in Docker) |
| `GMG_DB_MAX_SIZE_MB` | Database size limit; the oldest 10% of rows are pruned when exceeded | `50` |
| `GMG_GRILL_TEMP_MIN`, `GMG_GRILL_TEMP_MAX` | Grill target temperatures accepted (°F); others get `400` | `150`, `550` |
| `GMG_FOOD_TEMP_MIN`, `GMG_FOOD_TEMP_MAX` | Food probe target temperatures accepted (°F) | `0`, `300` |
| `GMG_ALERTS_AUDIO_ENABLED` | Play a sound in the browser when an alert fires (`true`/`false`); no sound files ship, see below | `false` |
| `GMG_ALERTS_UNREACHABLE_SECONDS` | Alert when the grill hasn't answered for this long while on or in fan mode; `0` turns the alert off | `60` |
| `GMG_ALERTS_TEMPDROP_DEGREES` | Alert when the grill falls this far below its target after reaching it; `0` turns the alert off | `50` |
| `GMG_ALERTS_TEMPDROP_MINUTES` | ...for at least this long | `5` |
| `GMG_ALERTS_IGNITION_DEGREES` | Alert when the grill hasn't warmed by this much after being turned on; `0` turns the alert off | `30` |
| `GMG_ALERTS_IGNITION_MINUTES` | ...within this long | `15` |
| `GMG_ALERTS_SLACK_WEBHOOKURL` | Slack incoming-webhook URL; Slack alerts are off when unset | none |
| `GMG_ALERTS_SLACK_CHANNEL` | Slack channel | `#alerts` |
| `GMG_ALERTS_SLACK_USERNAME` | Slack username | `grill-bot` |
| `GMG_CORS_ORIGINS` | Comma-separated origins (e.g. `https://grill.example.com`) allowed to call the API from a browser; `*` allows any. Same-origin requests are always allowed | none |
| `GMG_LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` |
| `PORT` | HTTP port | `3001` (`80` in Docker) |

No alert sounds are included. To use audio alerts, add `targetGrillTempReached.mp3`, `targetFoodTempReached.mp3` and `lowPelletAlarmActive.mp3` to `web/public/alerts/`, rebuild, and set `GMG_ALERTS_AUDIO_ENABLED=true`.

## Alerts

| Alert | When it fires |
|-------|---------------|
| Target Grill Temperature Reached | The grill reaches its target. Fires again for a new target, or after the grill has been turned off |
| Target Food Temperature Reached | The food probe reaches its target. Same re-arming as above |
| Grill Pellet Alarm | The grill reports low pellets. Fires again each time the alarm clears and comes back |
| Grill May Not Have Lit | The grill was turned on but hasn't warmed by `GMG_ALERTS_IGNITION_DEGREES` within `GMG_ALERTS_IGNITION_MINUTES`: usually a failed igniter or a fire pot full of ash. Not for a restart while still hot |
| Food Probe Disconnected | The food probe stops reporting while the grill is on and a food target is set. Fires again after it's plugged back in |
| Grill Temperature Dropping | After reaching its target, the grill stays `GMG_ALERTS_TEMPDROP_DEGREES` below it for `GMG_ALERTS_TEMPDROP_MINUTES`: usually the fire went out or the pellets jammed. Not during warm-up or after raising the target. Fires again after the grill recovers |
| Grill Not Responding | The grill was on (or cooling down in fan mode) and hasn't answered for `GMG_ALERTS_UNREACHABLE_SECONDS`. A grill that was off when it went quiet doesn't alert. "Grill Responding Again" follows when it comes back |

While the grill isn't answering, the dashboard dims the readings, shows when they were last updated and disables the controls.

## API

The running server documents its API at `/api` (also linked from the dashboard's bottom bar).

| Method | Path | |
|--------|------|-|
| `GET` | `/api/status` | Current grill status (JSON) |
| `PUT` | `/api/powertoggle`, `/api/poweron`, `/api/poweroff` | Power |
| `PUT` | `/api/temperature/grill/{°F}`, `/api/temperature/food/{°F}` | Set target temperatures |
| `GET` | `/api/limits` | Accepted target temperatures (JSON) |
| `GET` | `/api/temperature/history?since={unix seconds}` | Temperature history (JSON) |
| `GET` | `/api/connection` | Whether the grill is answering, and when it last did (JSON) |
| `GET` | `/api/events` | Live [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events): `status` on every good poll, `connection` after every poll attempt, `alert` when one fires |

Commands return `OK`. Browsers on another site can't send commands (`403`) unless their origin is in `GMG_CORS_ORIGINS`; this is a change from the original server, which let any web page control the grill. Clients that send no `Origin` header, like `curl` and scripts, are not affected. They return `400` with a message when the grill can't accept the command (for example, setting a temperature while it's off), and `500` with a message when the grill doesn't respond.

## Development

**Prerequisites:** Go 1.26+, Node 24+, GNU Make, Docker (for images and end-to-end tests). With [flox](https://flox.dev), `flox activate` provides Go, Node and Make.

```sh
make run     # build and run bin/gmg with your .env (http://localhost:3001)
make dev     # emulator + server (:3001) + Vite dev server with hot reload at http://localhost:3000
make test    # Go tests (with the race detector) and web UI tests
make check   # gofmt, go vet, svelte-check
make e2e     # build the Docker images and run browser end-to-end tests against them
make build   # bin/gmg (UI embedded) and bin/gmg-emulator
make help    # everything else
```

The tests and `make e2e` run against the emulator. Before relying on a change for an unattended cook, go through [`doc/real-grill-testing.md`](doc/real-grill-testing.md) on a real grill.

### Layout

| Path | |
|------|-|
| `cmd/gmg` | Server: polls the grill, records history, sends alerts, serves the API and UI |
| `cmd/gmg-emulator` | Grill emulator (UDP) |
| `internal/protocol` | GMG wire protocol: commands and the 36-byte status format (shared by client and emulator) |
| `internal/grill` | UDP client: discovery, retries, command validation |
| `internal/poller`, `alerts`, `store`, `api`, `config` | Server components |
| `web/` | Svelte + TypeScript UI, built with Vite and embedded into the server binary |
| `e2e/` | Playwright checks against the Docker images |
| `doc/decode.txt` | Notes and captures from decoding the grill's status format |
| `doc/real-grill-testing.md` | Checklist for testing against a real grill, and how to capture status frames |

## Tech stack
- **Server and emulator:** Go (standard library + pure-Go SQLite, no CGO)
- **UI:** Svelte 5, TypeScript, Vite, Chart.js
- **Tests:** Go testing, Vitest + Testing Library, Playwright

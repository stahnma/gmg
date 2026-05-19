# Green Mountain Grills App

### This project is a silly alternative to the Green Mountain Grills mobile app.

## Why?
Well, I like to Grill, and I often utilize the grill overnight when cooking large portions. Unfortunately the GMG mobile app is not a dependable source for alerting me when critical grill events occur (like, the grill is out of fuel/pellets).

## Features
1. Slack Alerts
1. Configurable browser alerts sounds
1. Auto connect/reconnect
1. REST API and JS Client for extensions
1. Timers
1. Grill controls (Power, food temp, grill temp)

## Quick start (local dev, no Docker)

The easiest path uses [flox](https://flox.dev) — it pins Node and the .NET SDK for you (see `.flox/env/manifest.toml`).

```bash
flox activate          # provides node, dotnet, gnumake; sets DOTNET_ROOT
make dev               # emulator (UDP/18080) + gmg-server (3001) + Vite (3000)
```

Then open <http://localhost:3000/>. The Vite dev server proxies `/api` and `/socket.io` to gmg-server, which in turn polls the emulator on UDP/18080 over the grill protocol. Ctrl-C tears down all three processes.

The emulator port is configurable (8080 is the *real* grill's port and is often already bound on dev machines):

```bash
make dev EMU_PORT=28080
```

Don't have flox? See [Manual local dev](#manual-local-dev-no-flox) below.

## Configuration

Server config layers, lowest priority first:

1. `src/gmg-server/config/default.json` — defaults
2. `src/gmg-server/config/custom-environment-variables.json` — maps env vars onto config keys
3. Environment variables — overrides at runtime

The interesting env vars:

| Var | Meaning | Default |
|---|---|---|
| `GMG_GRILL_HOST` | Grill (or emulator) IP | UDP broadcast / `255.255.255.255` |
| `GMG_GRILL_PORT` | Grill port | `8080` |
| `GMG_GRILL_TRIES` | Discovery attempts | `5` |
| `GMG_STATUS_POLLING_INTERVAL` | Poll interval (ms) | `5000` |
| `GMG_ALERTS_SLACK_WEBHOOKURL` | Slack webhook (optional) | unset |
| `GMG_EXTERNAL_PORT` | Host port mapped to the container (Docker only) | `80` |

If you want to set your own alert sounds, override the corresponding mp3 in `src/gmg-server/public/alerts/`.

## Manual local dev (no flox)

You need:

- Node.js **18+** (the UI build uses Vite 5)
- .NET SDK **11** (only if you want to run the emulator)
- `npm`, `make`

```bash
# one-time
(cd src/gmg-client   && npm install)
(cd src/gmg-server   && npm install)
(cd src/gmg-app      && npm install && npm run publish)   # builds UI -> ../gmg-server/public/app
(cd src/gmg-emulator && dotnet build)                     # optional, for the emulator

# run (two terminals)
# A) optional: emulator (skip if a real grill is on the network)
(cd src/gmg-emulator && dotnet run -- -p 18080)

# B) server (serves UI on / and API on /api)
cd src/gmg-server && GMG_GRILL_HOST=127.0.0.1 GMG_GRILL_PORT=18080 npm run start:dev
# open http://localhost:3001/
```

For UI hot-reload, run the Vite dev server in a third terminal instead of using the built bundle from `npm run publish`:

```bash
(cd src/gmg-app && npm start)   # http://localhost:3000/ with HMR
```

Vite proxies `/api` and `/socket.io` to `localhost:3001`, so leave the server running on `start:dev`.

`npm run start:release` (server) binds port **80** and needs sudo; `start:dev` binds **3001** and doesn't. Use `start:dev` for development.

## Setup and Run with Docker

```bash
cd src
docker build -t gmg .
docker run -it -p 80:80 \
  -e "GMG_GRILL_HOST=xx.xx.xx.xx" \
  -e "GMG_ALERTS_SLACK_WEBHOOKURL=https://your_slack_webhook_address" \
  gmg
```

Omit `GMG_ALERTS_SLACK_WEBHOOKURL` if you're not using Slack. The container cannot auto-discover the grill, so `GMG_GRILL_HOST` is required.

### Makefile (Docker workflow)

`make help` lists targets. The Docker-specific ones:

| Target | What it does |
|---|---|
| `make image` | Build the Docker image |
| `make image-nc` | Same, no cache |
| `make push-image` | Push to Docker Hub (set `DOCKER_HUB_USERNAME` first) |
| `make run` | `./start` — runs the image with env from your shell or `.env` |

For `make run`, set `GMG_GRILL_HOST` (and optionally `GMG_ALERTS_SLACK_WEBHOOKURL` / `GMG_EXTERNAL_PORT`) either inline (`GMG_GRILL_HOST=... make run`) or via a `.env` file in the repo root. If neither is set, `./start` will tell you what's missing — it no longer errors on a missing `.env` file.

## Setup and Run on a Raspberry Pi

The old hard-coded Node 14 install in this README is gone — modern Node releases don't ship for `armv6l` anyway. On a current Pi OS / Ubuntu (aarch64 or armv7l):

```bash
# install a current Node LTS (NodeSource works for Debian/Ubuntu/Raspbian)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

git clone https://github.com/Aenima4six2/gmg.git
cd gmg/src
./build.sh
cd gmg-server && sudo npm run start:release    # binds :80 (needs sudo)
```

Pi Zero / Pi 1 (armv6l) aren't supported by modern Node — use a Pi 2/3/4/5.

## Development and Debugging

- **Server** (`gmg-server`): start with `npm run start:dev` (port 3001, `NODE_ENV=development`, `DEBUG=src.*`). To attach a debugger, run `node --inspect ./bin/www` from `src/gmg-server/` and attach with your IDE. The provided `gmg.code-workspace` (VSCode) has `Debug (gmg-server)` configured.
- **UI** (`gmg-app`): `npm start` from `src/gmg-app/` launches the Vite dev server on port 3000 with HMR. The build is `npm run build` (writes directly into `../gmg-server/public/app`).
- **Emulator** (`gmg-emulator`): `dotnet run` from `src/gmg-emulator/` (defaults to UDP/8080; pass `-- -p N` for a different port). The csproj targets `net11.0`. The VSCode workspace includes a `Debug (gmg-emulator)` configuration.

`make dev` does all three in one terminal — see [Quick start](#quick-start-local-dev-no-docker).

## Preview
![alt text](assets/preview.jpg)

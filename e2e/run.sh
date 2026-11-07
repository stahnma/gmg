#!/usr/bin/env bash
# Builds the images and runs checks.mjs in the Playwright container against
# the server + emulator, with alert audio on and off, then checks that
# history survives a server restart. Screenshots go to e2e/out/<audio>/.
set -euo pipefail
cd "$(dirname "$0")/.."

PLAYWRIGHT_IMAGE=mcr.microsoft.com/playwright:v1.64.1-noble   # keep in sync with e2e/package.json
id=$$
net=gmg-e2e-$id
cleanup() { docker rm -f gmg-e2e-app-$id gmg-e2e-emu-$id >/dev/null 2>&1 || true; docker network rm $net >/dev/null 2>&1 || true; docker volume rm gmg-e2e-data-$id >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker build -q -t gmg:e2e . >/dev/null
docker build -q -t gmg-emulator:e2e -f Dockerfile.emulator . >/dev/null
docker network create $net >/dev/null
docker pull -q curlimages/curl:8.11.1 >/dev/null

start() { # audio
  docker rm -f gmg-e2e-app-$id gmg-e2e-emu-$id >/dev/null 2>&1 || true
  docker run -d --name gmg-e2e-emu-$id --network $net gmg-emulator:e2e >/dev/null
  docker run -d --name gmg-e2e-app-$id --network $net -v gmg-e2e-data-$id:/data \
    -e GMG_GRILL_HOST=gmg-e2e-emu-$id -e GMG_STATUS_POLLING_INTERVAL=2000 -e GMG_ALERTS_AUDIO_ENABLED=$1 gmg:e2e >/dev/null
}

status=0
for audio in on off; do
  mkdir -p e2e/out/$audio && chmod 777 e2e/out/$audio
  docker volume rm gmg-e2e-data-$id >/dev/null 2>&1 || true
  start "$([ $audio = on ] && echo true || echo false)"
  echo "== audio $audio"
  docker run --rm --network $net --ipc=host -v "$PWD/e2e:/e2e-src:ro" -v "$PWD/e2e/out/$audio:/out" $PLAYWRIGHT_IMAGE \
    bash -c 'cp /e2e-src/package.json /e2e-src/checks.mjs /tmp/ && cd /tmp && npm install --no-audit --no-fund --ignore-scripts -q >/dev/null 2>&1 && node checks.mjs http://gmg-e2e-app-'$id' /out '$audio || status=1
done

echo "== history survives a restart"
before=$(docker run --rm --network $net curlimages/curl:8.11.1 -s "http://gmg-e2e-app-$id/api/temperature/history?since=0")
docker restart gmg-e2e-app-$id >/dev/null; sleep 2
after=$(docker run --rm --network $net curlimages/curl:8.11.1 -s "http://gmg-e2e-app-$id/api/temperature/history?since=0")
if [ -n "$before" ] && [ "$before" != "[]" ] && [ "${after:0:${#before}-1}" = "${before:0:${#before}-1}" ]; then echo "ok    history kept across restart"; else echo "FAIL  history lost across restart"; status=1; fi
exit $status

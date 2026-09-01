#!/usr/bin/env bash

set -euo pipefail
umask 077

run_id="${GITHUB_RUN_ID:-$$}"
run_attempt="${GITHUB_RUN_ATTEMPT:-1}"
[[ "${run_id}" =~ ^[0-9]+$ ]]
[[ "${run_attempt}" =~ ^[0-9]+$ ]]

suffix="${run_id}-${run_attempt}"
lru_container="my-bbs-redis-lru-test-${suffix}"
persist_container="my-bbs-redis-persist-test-${suffix}"
persist_volume="my-bbs-redis-persist-test-${suffix}"
redis_image="redis:7.4-alpine"
compose_contract=""

cleanup() {
  docker rm -f -- "${lru_container}" "${persist_container}" >/dev/null 2>&1 || true
  docker volume rm -- "${persist_volume}" >/dev/null 2>&1 || true
  if [[ -n "${compose_contract}" && -f "${compose_contract}" ]]; then
    rm -f -- "${compose_contract}"
  fi
}
trap cleanup EXIT HUP INT TERM
cleanup

compose_contract="$(mktemp "${TMPDIR:-/tmp}/my-bbs-compose.XXXXXX.json")"
ADMIN_USERNAMES=ci docker compose config --format json > "${compose_contract}"
python3 - "${compose_contract}" <<'PY'
import json
import pathlib
import sys

config = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
services = config.get("services", {})


def command_options(service_name):
    command = services[service_name].get("command")
    if not isinstance(command, list) or not command or command[0] != "redis-server":
        raise SystemExit(f"{service_name} must use an explicit redis-server command")
    options = {}
    index = 1
    while index < len(command):
        name = command[index]
        if not isinstance(name, str) or not name.startswith("--") or index + 1 >= len(command):
            raise SystemExit(f"invalid {service_name} command near {name!r}")
        if name in options:
            raise SystemExit(f"duplicate {service_name} option {name}")
        options[name] = command[index + 1]
        index += 2
    return options


persist = command_options("redis-persist")
lru = command_options("redis-lru")

expected_persist = {
    "--appendonly": "yes",
    "--appendfsync": "everysec",
    "--maxmemory": "0",
    "--maxmemory-policy": "noeviction",
}
expected_lru = {
    "--save": "",
    "--appendonly": "no",
    "--maxmemory": "256mb",
    "--maxmemory-policy": "allkeys-lru",
    "--maxmemory-samples": "10",
}
for name, value in expected_persist.items():
    if persist.get(name) != value:
        raise SystemExit(f"redis-persist {name}={persist.get(name)!r}, expected {value!r}")
for name, value in expected_lru.items():
    if lru.get(name) != value:
        raise SystemExit(f"redis-lru {name}={lru.get(name)!r}, expected {value!r}")

persist_volumes = services["redis-persist"].get("volumes", [])
if not any(volume.get("target") == "/data" for volume in persist_volumes):
    raise SystemExit("redis-persist must mount a persistent /data volume")
if services["redis-lru"].get("volumes"):
    raise SystemExit("redis-lru must not mount persistent storage")

app_environment = services["app"].get("environment", {})
expected_environment = {
    "REDIS_PERSIST_ADDR": "redis-persist:6379",
    "REDIS_LRU_ADDR": "redis-lru:6379",
}
for name, value in expected_environment.items():
    if app_environment.get(name) != value:
        raise SystemExit(f"app {name}={app_environment.get(name)!r}, expected {value!r}")
PY

wait_for_redis() {
  local container="$1"
  local ready=false

  for _ in {1..60}; do
    if docker exec "${container}" redis-cli --raw PING 2>/dev/null | grep -qx 'PONG'; then
      ready=true
      break
    fi
    sleep 1
  done
  [[ "${ready}" == "true" ]]
}

config_value() {
  local container="$1"
  local key="$2"
  docker exec "${container}" redis-cli --raw CONFIG GET "${key}" | tail -n 1
}

stat_value() {
  local container="$1"
  local key="$2"
  docker exec "${container}" redis-cli --raw INFO stats \
    | awk -F: -v wanted="${key}" '$1 == wanted {gsub(/\r/, "", $2); print $2}'
}

docker volume create "${persist_volume}" >/dev/null

docker run --detach --name "${lru_container}" \
  "${redis_image}" \
  redis-server \
    --save '' \
    --appendonly no \
    --maxmemory 4mb \
    --maxmemory-policy allkeys-lru \
    --maxmemory-samples 10 >/dev/null

docker run --detach --name "${persist_container}" \
  --volume "${persist_volume}:/data" \
  "${redis_image}" \
  redis-server \
    --save '' \
    --appendonly yes \
    --appendfsync always \
    --maxmemory 4mb \
    --maxmemory-policy noeviction >/dev/null

wait_for_redis "${lru_container}"
wait_for_redis "${persist_container}"

[[ "$(config_value "${lru_container}" maxmemory-policy)" == "allkeys-lru" ]]
[[ "$(config_value "${lru_container}" maxmemory-samples)" == "10" ]]
[[ "$(config_value "${lru_container}" appendonly)" == "no" ]]
[[ -z "$(config_value "${lru_container}" save)" ]]
[[ "$(config_value "${persist_container}" maxmemory-policy)" == "noeviction" ]]
[[ "$(config_value "${persist_container}" appendonly)" == "yes" ]]

payload="$(dd if=/dev/zero bs=1024 count=48 2>/dev/null | base64 | tr -d '\n')"
[[ "${#payload}" -gt 64000 ]]

for i in {1..100}; do
  reply="$(docker exec "${lru_container}" redis-cli --raw SET "lru:${i}" "${payload}")"
  [[ "${reply}" == "OK" ]]
done

lru_evicted="$(stat_value "${lru_container}" evicted_keys)"
[[ "${lru_evicted}" =~ ^[0-9]+$ ]]
(( lru_evicted > 0 ))

reply="$(docker exec "${lru_container}" redis-cli --raw SET lru:restart-sentinel disposable)"
[[ "${reply}" == "OK" ]]
docker restart --time 15 "${lru_container}" >/dev/null
wait_for_redis "${lru_container}"
[[ "$(docker exec "${lru_container}" redis-cli --raw EXISTS lru:restart-sentinel)" == "0" ]]

protected_key="mybbs:v1:auth:revoked:policy-test"
protected_value="revoked"
reply="$(docker exec "${persist_container}" redis-cli --raw SET "${protected_key}" "${protected_value}" EX 3600)"
[[ "${reply}" == "OK" ]]

persist_rejected=false
for i in {1..100}; do
  reply="$(docker exec "${persist_container}" redis-cli --raw SET "persist:${i}" "${payload}" 2>&1 || true)"
  if [[ "${reply}" == OOM* ]]; then
    persist_rejected=true
    break
  fi
  [[ "${reply}" == "OK" ]]
done
[[ "${persist_rejected}" == "true" ]]

persist_evicted="$(stat_value "${persist_container}" evicted_keys)"
[[ "${persist_evicted}" == "0" ]]
[[ "$(docker exec "${persist_container}" redis-cli --raw GET "${protected_key}")" == "${protected_value}" ]]

docker stop --time 15 "${persist_container}" >/dev/null
docker rm -- "${persist_container}" >/dev/null

docker run --detach --name "${persist_container}" \
  --volume "${persist_volume}:/data" \
  "${redis_image}" \
  redis-server \
    --save '' \
    --appendonly yes \
    --appendfsync always \
    --maxmemory 4mb \
    --maxmemory-policy noeviction >/dev/null

wait_for_redis "${persist_container}"
[[ "$(docker exec "${persist_container}" redis-cli --raw GET "${protected_key}")" == "${protected_value}" ]]

printf 'REDIS_POLICY_OK lru_evicted=%s persist_evicted=%s\n' \
  "${lru_evicted}" "${persist_evicted}"

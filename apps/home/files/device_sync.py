import json
import os
import sys
import time
import urllib.error
import urllib.request

HA_URL = os.environ["HA_URL"]
HEADERS = {
    "Authorization": f"Bearer {os.environ['HA_TOKEN']}",
    "Content-Type": "application/json",
}
with open("/app/devices.json") as f:
    DEVICES = json.load(f)


def api(path, data=None):
    body = json.dumps(data).encode() if data is not None else None
    req = urllib.request.Request(HA_URL + path, data=body, headers=HEADERS)
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.load(resp)


def add_device(handler, label, steps):
    try:
        flow = api("/api/config/config_entries/flow", {"handler": handler, "show_advanced_options": False})
        result = {}
        for step in steps:
            result = api(f"/api/config/config_entries/flow/{flow['flow_id']}", step)
            if result.get("type") != "form":
                break
    except urllib.error.HTTPError as e:
        print(f"{label}: HTTP {e.code} {e.read().decode()[:200]}")
        return False
    kind, reason = result.get("type"), result.get("reason", "")
    if kind == "create_entry":
        print(f"{label}: added as '{result['result']['title']}'")
    elif reason.startswith("already_configured"):
        print(f"{label}: already configured")
    else:
        print(f"{label}: unexpected result: {kind} {reason} {result.get('errors')}")
        return False
    return True


for _ in range(24):
    try:
        api("/api/")
        break
    except (urllib.error.URLError, OSError):
        time.sleep(5)
else:
    sys.exit("home assistant not reachable")

failed = False
for host in DEVICES.get("shelly") or []:
    failed |= not add_device("shelly", host, [{"host": host, "port": 80}])
for dev in DEVICES.get("esphome") or []:
    with open(f"/secrets/esphome/{dev['name']}") as f:
        noise_psk = f.read().strip()
    failed |= not add_device(
        "esphome", dev["name"], [{"host": dev["host"], "port": 6053}, {"noise_psk": noise_psk}]
    )

sys.exit(1 if failed else 0)

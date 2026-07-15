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


for _ in range(24):
    try:
        api("/api/")
        break
    except (urllib.error.URLError, OSError):
        time.sleep(5)
else:
    sys.exit("home assistant not reachable")

failed = False
for host in DEVICES:
    try:
        flow = api("/api/config/config_entries/flow", {"handler": "shelly", "show_advanced_options": False})
        result = api(f"/api/config/config_entries/flow/{flow['flow_id']}", {"host": host, "port": 80})
    except urllib.error.HTTPError as e:
        print(f"{host}: HTTP {e.code} {e.read().decode()[:200]}")
        failed = True
        continue
    kind, reason = result.get("type"), result.get("reason", "")
    if kind == "create_entry":
        print(f"{host}: added as '{result['result']['title']}'")
    elif reason == "already_configured":
        print(f"{host}: already configured")
    else:
        print(f"{host}: unexpected result: {kind} {reason} {result.get('errors')}")
        failed = True

sys.exit(1 if failed else 0)

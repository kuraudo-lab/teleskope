"""Verify report serving from the binary alone, with no Node or cluster access."""
import json
import os
import queue
import subprocess
import threading
import urllib.request
from pathlib import Path


def verify_embedded_report(binary, directory):
    directory = Path(directory)
    snapshot = directory / "snapshot.json"
    collected = "2026-09-01T00:00:00Z"
    snapshot.write_text(json.dumps({
        "schemaVersion": "teleskope.io/snapshot/v1alpha1", "collectedAt": collected,
        "source": {"mode": "recorded"},
        "kubernetes": {"context": "package-smoke", "nodes": [{"kind": "Node", "name": "node-demo"}]},
    }), encoding="utf-8")
    env = {**os.environ, "PATH": "", "KUBECONFIG": str(directory / "no-kubeconfig"),
           "AWS_CONFIG_FILE": str(directory / "no-aws-config"),
           "AWS_SHARED_CREDENTIALS_FILE": str(directory / "no-aws-credentials"),
           "AWS_EC2_METADATA_DISABLED": "true"}
    process = subprocess.Popen([str(Path(binary).resolve()), "serve", "snapshot", str(snapshot),
                                "--listen", "127.0.0.1:0"], cwd=directory, env=env,
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    output = queue.Queue()
    threading.Thread(target=lambda: output.put(process.stdout.readline()), daemon=True).start()
    try:
        line = output.get(timeout=20).strip()
        if not line.startswith("Recorded inventory: http://127.0.0.1:"):
            raise ValueError(f"Report server did not start: {line}")
        url = line.split("Recorded inventory: ", 1)[1]
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        with opener.open(url, timeout=10) as response:
            html = response.read().decode("utf-8")
        if 'id="ui-root"' not in html or '"analysisEnabled":false' not in html:
            raise ValueError("Embedded report boot contract missing")
        if '<script src=' in html or '<link rel="stylesheet"' in html:
            raise ValueError("Report depends on external assets")
        with opener.open(url + "/api/export/snapshot.json", timeout=10) as response:
            data = json.load(response)
        if data["kubernetes"]["context"] != "package-smoke" or data["collectedAt"] != collected:
            raise ValueError("Replay did not retain archived evidence")
    finally:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
        process.stdout.close()
        process.stderr.close()

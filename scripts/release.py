"""Validate release archives and smoke-test the native binary without cloud access."""

import hashlib
import json
import platform
import subprocess
import sys
import tarfile
import tempfile
import zipfile
from pathlib import Path


def archive_name(version, os_name, arch):
    extension = "zip" if os_name == "windows" else "tar.gz"
    return f"teleskope_{version}_{os_name}_{arch}.{extension}"


def verify(directory, version):
    expected = {
        archive_name(version, os_name, arch)
        for os_name in ("linux", "darwin", "windows")
        for arch in ("amd64", "arm64")
    }
    checksums = {}
    for line in (directory / "checksums.txt").read_text(encoding="utf-8").splitlines():
        digest, name = line.split()
        if name in checksums:
            raise ValueError(f"Duplicate checksum: {name}")
        checksums[name] = digest
    if set(checksums) != expected:
        raise ValueError("Checksum manifest must contain exactly the six target archives")
    actual = {p.name for p in directory.iterdir() if p.name.endswith((".tar.gz", ".zip"))}
    if actual != expected:
        raise ValueError("Archive set does not match the six release targets")
    for name, digest in checksums.items():
        with (directory / name).open("rb") as stream:
            if hashlib.file_digest(stream, "sha256").hexdigest() != digest:
                raise ValueError(f"Checksum mismatch: {name}")


def smoke(directory, version, os_name, arch):
    verify(directory, version)
    native_arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}
    if platform.system().lower() != os_name or native_arch.get(platform.machine().lower()) != arch:
        raise ValueError("Smoke test must run on the target OS and architecture")
    archive = directory / archive_name(version, os_name, arch)
    binary_name = "teleskope.exe" if os_name == "windows" else "teleskope"
    with tempfile.TemporaryDirectory(prefix="teleskope-smoke-") as temporary:
        root = Path(temporary)
        # Extract only the expected binary; do not trust archive paths.
        if os_name == "windows":
            with zipfile.ZipFile(archive) as package:
                data = package.read(binary_name)
        else:
            with tarfile.open(archive) as package:
                data = package.extractfile(binary_name).read()
        binary = root / binary_name
        binary.write_bytes(data)
        binary.chmod(0o755)

        def run(*args):
            return subprocess.run([str(binary), *args], check=True, capture_output=True, text=True, timeout=30).stdout

        if run("--version").strip() != f"teleskope {version}":
            raise ValueError("Incorrect embedded version")
        if "Usage:" not in run("--help"):
            raise ValueError("Missing CLI help")
        output = run("scan", "k8s", "--kubeconfig", str(root / "missing-config"),
                     "--timeout", "1s", "--output-dir", str(root / "reports"))
        if not output.startswith("report "):
            raise ValueError(f"Unexpected report output: {output}")
        report = Path(output.removeprefix("report ").strip())
        for name in ("snapshot.json", "source.json", "kubernetes.json", "coverage.json", "summary.md", "index.html"):
            if not (report / name).is_file():
                raise ValueError(f"Missing report artifact: {name}")
        snapshot = json.loads((report / "snapshot.json").read_text(encoding="utf-8"))
        if snapshot["source"]["version"] != version:
            raise ValueError("Report version does not match the release")
        html = (report / "index.html").read_text(encoding="utf-8")
        if "__TELESKOPE_SNAPSHOT_JSON__" in html or "<html" not in html.lower():
            raise ValueError("Embedded HTML report was not rendered")
    print(f"Verified {os_name}/{arch} {version}")


if __name__ == "__main__":
    command, directory, version, *target = sys.argv[1:]
    if command == "verify" and not target:
        verify(Path(directory), version)
    elif command == "smoke" and len(target) == 2:
        smoke(Path(directory), version, *target)
    else:
        sys.exit("Usage: release.py verify DIR VERSION | smoke DIR VERSION OS ARCH")

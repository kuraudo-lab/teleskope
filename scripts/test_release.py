"""Release licensing contract: every target archive must carry the approved text."""
import hashlib
import io
import tarfile
import tempfile
import unittest
import zipfile
from pathlib import Path

import release


class LicenseArchiveTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.license = (Path(__file__).resolve().parents[1] / "LICENSE").read_bytes()

    def package(self, os_name, arch="arm64", content=None, include=True):
        path = self.root / release.archive_name("1.0.0", os_name, arch)
        content = self.license if content is None else content
        if os_name == "windows":
            with zipfile.ZipFile(path, "w") as archive:
                if include:
                    archive.writestr("LICENSE", content)
        else:
            with tarfile.open(path, "w:gz") as archive:
                if include:
                    member = tarfile.TarInfo("LICENSE")
                    member.size = len(content)
                    archive.addfile(member, io.BytesIO(content))
        return path

    def manifest(self):
        files = sorted(p for p in self.root.iterdir() if p.name != "checksums.txt")
        (self.root / "checksums.txt").write_text("".join(
            f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n" for p in files
        ), encoding="utf-8")

    def test_all_six_archives_include_license(self):
        for os_name in ("linux", "darwin", "windows"):
            for arch in ("amd64", "arm64"):
                self.package(os_name, arch)
        self.manifest()
        release.verify(self.root, "1.0.0")
        # Correct checksums cannot hide a package missing its license.
        self.package("windows", include=False)
        self.manifest()
        with self.assertRaisesRegex(ValueError, "Missing LICENSE"):
            release.verify(self.root, "1.0.0")

    def test_missing_and_modified_text_rejected_in_both_formats(self):
        for os_name in ("linux", "windows"):
            with self.subTest(os_name=os_name):
                with self.assertRaisesRegex(ValueError, "Missing LICENSE"):
                    release.verify_license(self.package(os_name, include=False))
                with self.assertRaisesRegex(ValueError, "does not match"):
                    release.verify_license(self.package(os_name, content=b"Apache-2.0\n"))

    def test_crlf_is_accepted(self):
        for os_name in ("linux", "windows"):
            content = self.license.replace(b"\r\n", b"\n").replace(b"\n", b"\r\n")
            release.verify_license(self.package(os_name, content=content))

    def test_tar_symlink_is_rejected(self):
        path = self.root / "symlink.tar.gz"
        with tarfile.open(path, "w:gz") as archive:
            member = tarfile.TarInfo("LICENSE")
            member.type = tarfile.SYMTYPE
            member.linkname = "/outside/LICENSE"
            archive.addfile(member)
        with self.assertRaisesRegex(ValueError, "regular file"):
            release.verify_license(path)


if __name__ == "__main__":
    unittest.main()

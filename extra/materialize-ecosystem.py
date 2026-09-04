from __future__ import annotations

import base64
import hashlib
from pathlib import Path, PurePosixPath
import tarfile

ROOT = Path(__file__).resolve().parents[1]
PART_GLOB = "ecosystem-source.tar.gz.b64.part-*"
EXPECTED_SHA256 = "8a4c62a53c61a622755af1bfa9eb4b2be71b6e1840de6dab52760082532459cd"
ALLOWED_FILES = {
    ".eslintignore",
    ".github/workflows/70-manager.yml",
    ".github/workflows/71-deploy.yml",
    "docs/ecosystem/IMPLEMENTATION-STATUS.md",
}
ALLOWED_PREFIXES = ("dockge-manager/", "dockge-deploy/")


def normalized_name(member: tarfile.TarInfo) -> str:
    name = member.name.removeprefix("./")
    path = PurePosixPath(name)
    if not name or path.is_absolute() or ".." in path.parts:
        raise RuntimeError(f"unsafe archive path: {member.name}")
    if member.issym() or member.islnk():
        raise RuntimeError(f"links are not allowed in bootstrap archive: {member.name}")
    return name


def allowed(name: str) -> bool:
    return name in ALLOWED_FILES or any(name.startswith(prefix) for prefix in ALLOWED_PREFIXES)


def main() -> None:
    parts = sorted((ROOT / "extra").glob(PART_GLOB))
    if not parts:
        raise RuntimeError("bootstrap source parts not found")
    encoded = "".join(part.read_text(encoding="ascii").strip() for part in parts)
    payload = base64.b64decode(encoded, validate=True)
    digest = hashlib.sha256(payload).hexdigest()
    if digest != EXPECTED_SHA256:
        raise RuntimeError(f"bootstrap archive checksum mismatch: {digest}")

    archive = ROOT / "extra" / ".ecosystem-source.tar.gz"
    archive.write_bytes(payload)
    try:
        with tarfile.open(archive, mode="r:gz") as bundle:
            members = bundle.getmembers()
            files = [normalized_name(member) for member in members if member.isfile()]
            unexpected = [name for name in files if not allowed(name)]
            if unexpected:
                raise RuntimeError(f"unexpected bootstrap files: {unexpected}")
            required = {".github/workflows/70-manager.yml", ".github/workflows/71-deploy.yml"}
            if not required.issubset(files):
                raise RuntimeError("bootstrap archive is missing ecosystem workflows")
            if not any(name.startswith("dockge-manager/") for name in files):
                raise RuntimeError("bootstrap archive is missing Dockge Manager")
            if not any(name.startswith("dockge-deploy/") for name in files):
                raise RuntimeError("bootstrap archive is missing Dockge Deploy")
            for member in members:
                name = normalized_name(member)
                if member.isdir():
                    if allowed(name + "/") or any(prefix.startswith(name + "/") for prefix in ALLOWED_PREFIXES):
                        (ROOT / name).mkdir(parents=True, exist_ok=True)
                    continue
                if not member.isfile() or not allowed(name):
                    continue
                target = ROOT / name
                target.parent.mkdir(parents=True, exist_ok=True)
                source = bundle.extractfile(member)
                if source is None:
                    raise RuntimeError(f"cannot read archive member: {name}")
                target.write_bytes(source.read())
                target.chmod(member.mode & 0o777)
    finally:
        archive.unlink(missing_ok=True)

    for part in parts:
        part.unlink(missing_ok=True)
    (ROOT / "extra" / "ecosystem-bootstrap.tar.gz").unlink(missing_ok=True)
    (ROOT / ".github" / "workflows" / "98-clean-v160-agent-assets.yml").unlink(missing_ok=True)
    print("Dockge Manager/Deploy source materialized and verified.")


if __name__ == "__main__":
    main()

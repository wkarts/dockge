from __future__ import annotations

import time
from dataclasses import dataclass
from typing import Any, Protocol


class PsClient(Protocol):
    def ps(self, name: str) -> dict: ...


@dataclass(slots=True)
class StackVerificationError(RuntimeError):
    reason: str
    payload: dict

    def __str__(self) -> str:
        return self.reason


def _container_rows(payload: dict) -> list[dict[str, Any]]:
    containers = payload.get("containers")
    if isinstance(containers, list):
        return [row for row in containers if isinstance(row, dict)]
    if isinstance(containers, dict):
        if not containers:
            return []
        # Docker Compose versions normally return an array; tolerate a single
        # object for compatibility with older JSON output shapes.
        if any(key in containers for key in ("State", "state", "Name", "name", "Service", "service")):
            return [containers]
        return [row for row in containers.values() if isinstance(row, dict)]
    return []


def assess_stack_ps(payload: dict) -> tuple[bool, str]:
    rows = _container_rows(payload)
    if not rows:
        return False, "stack_has_no_containers"

    for row in rows:
        state = str(row.get("State") or row.get("state") or "").strip().lower()
        if state != "running":
            name = str(row.get("Name") or row.get("name") or row.get("Service") or row.get("service") or "container")
            return False, f"{name}_state_{state or 'unknown'}"
        health = str(row.get("Health") or row.get("health") or "").strip().lower()
        if health not in {"", "healthy"}:
            name = str(row.get("Name") or row.get("name") or row.get("Service") or row.get("service") or "container")
            return False, f"{name}_health_{health}"
    return True, "healthy"


def verify_stack(client: PsClient, stack_name: str, attempts: int, interval_seconds: float) -> dict:
    last_payload: dict = {}
    last_reason = "verification_not_started"
    for attempt in range(max(1, attempts)):
        last_payload = client.ps(stack_name)
        ok, last_reason = assess_stack_ps(last_payload)
        if ok:
            return last_payload
        if attempt + 1 < attempts and interval_seconds > 0:
            time.sleep(interval_seconds)
    raise StackVerificationError(last_reason, last_payload)

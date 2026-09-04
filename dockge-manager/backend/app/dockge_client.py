from __future__ import annotations

from copy import deepcopy
from dataclasses import dataclass
import time
from typing import Any
from urllib.parse import urlparse

import httpx

from .config import get_settings


@dataclass(slots=True)
class DockgeRequestError(RuntimeError):
    status_code: int
    detail: Any

    def __str__(self) -> str:
        return f"Dockge API returned HTTP {self.status_code}: {self.detail}"


def validate_target_url(value: str) -> str:
    settings = get_settings()
    url = value.rstrip("/")
    parsed = urlparse(url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError("Dockge target must be an absolute http(s) URL")
    if parsed.scheme == "http" and not settings.allow_http_targets:
        raise ValueError("HTTP targets are disabled; use HTTPS or set DOCKGE_MANAGER_ALLOW_HTTP_TARGETS=true")
    return url


def sanitize_stack_payload(payload: Any) -> Any:
    """Remove environment secrets before Dockge responses reach the browser."""
    clean = deepcopy(payload)

    def sanitize_stack(value: Any) -> None:
        if not isinstance(value, dict):
            return
        value.pop("composeENV", None)
        value.pop("compose_env", None)

    if isinstance(clean, dict):
        sanitize_stack(clean)
        stacks = clean.get("stacks")
        if isinstance(stacks, list):
            for stack in stacks:
                sanitize_stack(stack)
    return clean


def runtime_health(payload: Any) -> tuple[bool, dict[str, Any]]:
    if not isinstance(payload, dict):
        return False, {"error": "invalid_ps_payload"}
    containers = payload.get("containers")
    if isinstance(containers, dict):
        containers = [containers] if containers else []
    if not isinstance(containers, list) or not containers:
        return False, {"error": "no_running_containers", "containers": 0}

    failures: list[dict[str, str]] = []
    for item in containers:
        if not isinstance(item, dict):
            failures.append({"container": "unknown", "reason": "invalid_container_payload"})
            continue
        name = str(item.get("Name") or item.get("name") or item.get("Service") or "unknown")
        state = str(item.get("State") or item.get("state") or "").lower()
        health = str(item.get("Health") or item.get("health") or "").lower()
        if state and state != "running":
            failures.append({"container": name, "reason": f"state={state}"})
        elif health and health != "healthy":
            failures.append({"container": name, "reason": f"health={health}"})

    return not failures, {
        "containers": len(containers),
        "failures": failures,
    }


class DockgeClient:
    def __init__(self, base_url: str, token: str, verify_tls: bool = True) -> None:
        self.base_url = validate_target_url(base_url)
        self.api_base = self.base_url + "/api/v1/automation"
        self.token = token
        self.verify_tls = verify_tls

    def _request(
        self,
        method: str,
        path: str,
        *,
        json: dict | None = None,
        params: dict | None = None,
        idempotency_key: str | None = None,
    ) -> Any:
        headers = {"Authorization": f"Bearer {self.token}"}
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        try:
            with httpx.Client(timeout=httpx.Timeout(45.0), verify=self.verify_tls) as client:
                response = client.request(
                    method,
                    self.api_base + path,
                    headers=headers,
                    json=json,
                    params=params,
                )
        except httpx.HTTPError as exc:
            raise DockgeRequestError(503, {"error": "dockge_unreachable", "message": str(exc)}) from exc

        try:
            payload: Any = response.json()
        except ValueError:
            payload = {"raw": response.text[-4096:]}
        if response.status_code >= 400:
            raise DockgeRequestError(response.status_code, payload)
        return payload

    def health(self) -> dict:
        return self._request("GET", "/health")

    def info(self) -> dict:
        return self._request("GET", "/info")

    def stacks(self) -> dict:
        return sanitize_stack_payload(self._request("GET", "/stacks"))

    def stack(self, name: str) -> dict:
        return sanitize_stack_payload(self._request("GET", f"/stacks/{name}"))

    def ps(self, name: str, attempts: int = 12, interval_seconds: float = 2.0) -> dict:
        last_payload: dict[str, Any] = {}
        last_detail: dict[str, Any] = {"error": "verification_not_started"}
        for attempt in range(max(attempts, 1)):
            payload = self._request("GET", f"/stacks/{name}/ps")
            if isinstance(payload, dict):
                last_payload = payload
            healthy, detail = runtime_health(payload)
            last_detail = detail
            if healthy:
                return last_payload
            if attempt + 1 < attempts:
                time.sleep(max(interval_seconds, 0.0))
        raise DockgeRequestError(
            502,
            {
                "error": "stack_health_verification_failed",
                "stack": name,
                "verification": last_detail,
            },
        )

    def logs(self, name: str, tail: int = 200) -> dict:
        return self._request("GET", f"/stacks/{name}/logs", params={"tail": tail})

    def apply_stack(self, name: str, body: dict, idempotency_key: str) -> dict:
        return self._request("PUT", f"/stacks/{name}", json=body, idempotency_key=idempotency_key)

    def action(self, name: str, action: str, idempotency_key: str) -> dict:
        return self._request("POST", f"/stacks/{name}/actions/{action}", json={}, idempotency_key=idempotency_key)

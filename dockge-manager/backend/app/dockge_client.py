from __future__ import annotations

from dataclasses import dataclass
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
            with httpx.Client(
                timeout=httpx.Timeout(45.0),
                verify=self.verify_tls,
                follow_redirects=False,
            ) as client:
                response = client.request(
                    method,
                    self.api_base + path,
                    headers=headers,
                    json=json,
                    params=params,
                )
        except httpx.HTTPError as exc:
            raise DockgeRequestError(503, {"error": "dockge_unreachable", "message": str(exc)}) from exc

        if 300 <= response.status_code < 400:
            raise DockgeRequestError(response.status_code, {"error": "dockge_redirect_blocked"})
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
        return self._request("GET", "/stacks")

    def stack(self, name: str) -> dict:
        return self._request("GET", f"/stacks/{name}")

    def ps(self, name: str) -> dict:
        return self._request("GET", f"/stacks/{name}/ps")

    def logs(self, name: str, tail: int = 200) -> dict:
        return self._request("GET", f"/stacks/{name}/logs", params={"tail": tail})

    def apply_stack(self, name: str, body: dict, idempotency_key: str) -> dict:
        return self._request("PUT", f"/stacks/{name}", json=body, idempotency_key=idempotency_key)

    def delete_stack(self, name: str, idempotency_key: str) -> dict:
        return self._request("DELETE", f"/stacks/{name}", idempotency_key=idempotency_key)

    def action(self, name: str, action: str, idempotency_key: str) -> dict:
        return self._request("POST", f"/stacks/{name}/actions/{action}", json={}, idempotency_key=idempotency_key)

import asyncio
from datetime import datetime, timezone

from sqlalchemy import select

from .config import get_settings
from .db import session_scope
from .dockge_client import DockgeRequestError
from .models import DockgeTarget, HealthSnapshot
from .security import SecretBox
from .service import client_for


def poll_once() -> None:
    settings = get_settings()
    box = SecretBox(settings.fernet_key)
    with session_scope() as db:
        targets = list(db.scalars(select(DockgeTarget).where(DockgeTarget.enabled.is_(True))))
        for target in targets:
            try:
                data = client_for(target, box).health()
                target.last_seen_at = datetime.now(timezone.utc)
                target.last_version = str(data.get("version") or "") or target.last_version
                db.add(
                    HealthSnapshot(
                        target_id=target.id,
                        ok=bool(data.get("ok", True)),
                        version=target.last_version,
                        detail_json=data,
                    )
                )
            except DockgeRequestError as exc:
                db.add(
                    HealthSnapshot(
                        target_id=target.id,
                        ok=False,
                        version=target.last_version,
                        detail_json={"status": exc.status_code, "detail": exc.detail},
                    )
                )


async def monitoring_loop(stop: asyncio.Event) -> None:
    settings = get_settings()
    while not stop.is_set():
        try:
            await asyncio.to_thread(poll_once)
        except Exception:
            # Monitoring must never terminate the API process. Failures are
            # represented by snapshots whenever they are target-specific.
            pass
        try:
            await asyncio.wait_for(stop.wait(), timeout=max(settings.health_poll_seconds, 10))
        except TimeoutError:
            continue

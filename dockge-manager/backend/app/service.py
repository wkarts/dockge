from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

from sqlalchemy.orm import Session

from .dockge_client import DockgeClient, DockgeRequestError
from .models import AuditEvent, DockgeTarget, Operation
from .security import SecretBox


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def client_for(target: DockgeTarget, secret_box: SecretBox) -> DockgeClient:
    return DockgeClient(target.base_url, secret_box.decrypt(target.credential.secret_ciphertext), target.verify_tls)


def audit(
    db: Session,
    actor: str,
    event_type: str,
    *,
    target_id: str | None = None,
    resource: str = "",
    payload: dict | None = None,
) -> None:
    db.add(
        AuditEvent(
            actor=actor,
            event_type=event_type,
            target_id=target_id,
            resource=resource,
            payload=payload or {},
        )
    )


def run_mutation(
    db: Session,
    target: DockgeTarget,
    actor: str,
    stack_name: str,
    action: str,
    callback,
) -> dict:
    key = str(uuid4())
    operation = Operation(
        target_id=target.id,
        stack_name=stack_name,
        action=action,
        idempotency_key=key,
        status="RUNNING",
    )
    db.add(operation)
    db.flush()
    try:
        result = callback(key)
        operation.status = "SUCCEEDED"
        operation.http_status = 200
        operation.response_json = result if isinstance(result, dict) else {"result": result}
        audit(db, actor, f"operation.{action}.succeeded", target_id=target.id, resource=stack_name)
        return result
    except DockgeRequestError as exc:
        operation.status = "FAILED"
        operation.http_status = exc.status_code
        operation.response_json = exc.detail if isinstance(exc.detail, dict) else {"detail": str(exc.detail)}
        audit(
            db,
            actor,
            f"operation.{action}.failed",
            target_id=target.id,
            resource=stack_name,
            payload={"status": exc.status_code},
        )
        raise
    finally:
        operation.completed_at = now_utc()
        db.flush()

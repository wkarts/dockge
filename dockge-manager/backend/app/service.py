from __future__ import annotations

import time
from datetime import datetime, timezone
from uuid import uuid4

from sqlalchemy.orm import Session

from .config import get_settings
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


def _retryable_mutation_error(exc: DockgeRequestError) -> bool:
    if exc.idempotency_in_doubt:
        return False
    if exc.is_transport_error:
        return True
    # O Dockge Core mantém a reserva idempotente aberta em falhas 5xx. Uma
    # única repetição com a MESMA chave converte o caso ambíguo em replay ou
    # em idempotency_result_in_doubt, sem executar a intenção duas vezes.
    return exc.status_code >= 500 and exc.may_have_mutated


def _operation_error_payload(exc: DockgeRequestError, retries_used: int) -> dict:
    if isinstance(exc.detail, dict):
        payload = dict(exc.detail)
    else:
        payload = {"detail": str(exc.detail)}
    payload["_manager"] = {
        "retries_used": retries_used,
        "transport_stage": exc.transport_stage,
        "may_have_mutated": exc.may_have_mutated,
    }
    return payload


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

    settings = get_settings()
    retries_used = 0
    try:
        while True:
            try:
                result = callback(key)
                operation.status = "SUCCEEDED"
                operation.http_status = 200
                response = result if isinstance(result, dict) else {"result": result}
                operation.response_json = dict(response)
                if retries_used:
                    operation.response_json["_manager"] = {
                        "retries_used": retries_used,
                        "reconciled_with_same_idempotency_key": True,
                    }
                audit(
                    db,
                    actor,
                    f"operation.{action}.succeeded",
                    target_id=target.id,
                    resource=stack_name,
                    payload={"retries_used": retries_used},
                )
                return result
            except DockgeRequestError as exc:
                if retries_used < settings.mutation_retry_attempts and _retryable_mutation_error(exc):
                    retries_used += 1
                    audit(
                        db,
                        actor,
                        f"operation.{action}.retry",
                        target_id=target.id,
                        resource=stack_name,
                        payload={
                            "attempt": retries_used,
                            "status": exc.status_code,
                            "transport_stage": exc.transport_stage,
                            "same_idempotency_key": True,
                        },
                    )
                    db.flush()
                    if settings.mutation_retry_delay_seconds > 0:
                        time.sleep(settings.mutation_retry_delay_seconds * retries_used)
                    continue

                operation.status = "IN_DOUBT" if exc.may_have_mutated else "FAILED"
                operation.http_status = exc.status_code
                operation.response_json = _operation_error_payload(exc, retries_used)
                suffix = "in_doubt" if exc.may_have_mutated else "failed"
                audit(
                    db,
                    actor,
                    f"operation.{action}.{suffix}",
                    target_id=target.id,
                    resource=stack_name,
                    payload={
                        "status": exc.status_code,
                        "retries_used": retries_used,
                        "may_have_mutated": exc.may_have_mutated,
                    },
                )
                raise
    finally:
        operation.completed_at = now_utc()
        db.flush()

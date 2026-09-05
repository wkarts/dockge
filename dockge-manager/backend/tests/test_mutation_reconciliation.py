import base64
import os
from types import SimpleNamespace

import pytest

os.environ.setdefault("DOCKGE_MANAGER_DATABASE_URL", "sqlite:///./test-manager.db")
os.environ.setdefault("DOCKGE_MANAGER_JWT_SECRET", "test-secret-that-is-at-least-thirty-two-characters-long")
os.environ.setdefault("DOCKGE_MANAGER_FERNET_KEY", base64.urlsafe_b64encode(b"x" * 32).decode("ascii"))
os.environ.setdefault("DOCKGE_MANAGER_ADMIN_PASSWORD", "test-password")
os.environ.setdefault("DOCKGE_MANAGER_HEALTH_POLL_ENABLED", "false")
os.environ.setdefault("DOCKGE_MANAGER_ALLOW_HTTP_TARGETS", "true")

from app.api import restore_runtime_snapshot  # noqa: E402
from app.dockge_client import DockgeRequestError  # noqa: E402
from app.models import Operation  # noqa: E402
from app import service  # noqa: E402


class FakeDb:
    def __init__(self) -> None:
        self.items: list[object] = []

    def add(self, value: object) -> None:
        self.items.append(value)

    def flush(self) -> None:
        return None


def operation_from(db: FakeDb) -> Operation:
    return next(item for item in db.items if isinstance(item, Operation))


def test_run_mutation_retries_with_exact_same_idempotency_key(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        service,
        "get_settings",
        lambda: SimpleNamespace(mutation_retry_attempts=2, mutation_retry_delay_seconds=0),
    )
    db = FakeDb()
    target = SimpleNamespace(id="target-1")
    keys: list[str] = []

    def callback(key: str) -> dict:
        keys.append(key)
        if len(keys) == 1:
            raise DockgeRequestError(
                503,
                {"error": "dockge_transport_uncertain"},
                transport_stage="uncertain",
                may_have_mutated=True,
            )
        return {"ok": True, "replayed": True}

    result = service.run_mutation(db, target, "admin@example.com", "demo", "deploy.apply", callback)

    assert result["ok"] is True
    assert len(keys) == 2
    assert keys[0] == keys[1]
    operation = operation_from(db)
    assert operation.status == "SUCCEEDED"
    assert operation.response_json["_manager"]["retries_used"] == 1
    assert operation.response_json["_manager"]["reconciled_with_same_idempotency_key"] is True


def test_run_mutation_stops_and_marks_in_doubt(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        service,
        "get_settings",
        lambda: SimpleNamespace(mutation_retry_attempts=3, mutation_retry_delay_seconds=0),
    )
    db = FakeDb()
    target = SimpleNamespace(id="target-2")
    keys: list[str] = []

    def callback(key: str) -> dict:
        keys.append(key)
        if len(keys) == 1:
            raise DockgeRequestError(
                503,
                {"error": "dockge_transport_uncertain"},
                transport_stage="uncertain",
                may_have_mutated=True,
            )
        raise DockgeRequestError(
            409,
            {"error": "idempotency_result_in_doubt"},
            may_have_mutated=True,
        )

    with pytest.raises(DockgeRequestError) as raised:
        service.run_mutation(db, target, "admin@example.com", "demo", "deploy.apply", callback)

    assert raised.value.idempotency_in_doubt is True
    assert len(keys) == 2
    assert keys[0] == keys[1]
    operation = operation_from(db)
    assert operation.status == "IN_DOUBT"
    assert operation.response_json["_manager"]["may_have_mutated"] is True


def test_connect_stage_failure_is_definitive_after_retries(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        service,
        "get_settings",
        lambda: SimpleNamespace(mutation_retry_attempts=1, mutation_retry_delay_seconds=0),
    )
    db = FakeDb()
    target = SimpleNamespace(id="target-3")
    calls = 0

    def callback(key: str) -> dict:
        nonlocal calls
        assert key
        calls += 1
        raise DockgeRequestError(
            503,
            {"error": "dockge_unreachable", "transport_stage": "connect"},
            transport_stage="connect",
            may_have_mutated=False,
        )

    with pytest.raises(DockgeRequestError) as raised:
        service.run_mutation(db, target, "admin@example.com", "demo", "restart", callback)

    assert calls == 2
    assert raised.value.may_have_mutated is False
    assert operation_from(db).status == "FAILED"


def test_restore_absent_snapshot_does_not_delete_when_request_never_mutated() -> None:
    class Client:
        def stack(self, name: str) -> dict:
            assert name == "demo"
            raise DockgeRequestError(404, {"error": "stack_not_found"})

        def delete_stack(self, name: str, key: str) -> dict:  # pragma: no cover - must never run
            raise AssertionError("delete must not run when the stack is already absent")

    snapshot = SimpleNamespace(existed=False, restored_at=None)
    result = restore_runtime_snapshot(
        FakeDb(),
        SimpleNamespace(email="admin@example.com"),
        SimpleNamespace(id="target-4"),
        SimpleNamespace(stack_name="demo"),
        Client(),
        snapshot,
    )

    assert result == {"restored": "stack_absence_already_present"}
    assert snapshot.restored_at is not None


def test_restore_absent_snapshot_refuses_to_delete_external_stack() -> None:
    class Client:
        def stack(self, name: str) -> dict:
            assert name == "demo"
            return {"name": "demo", "api_managed": False}

        def delete_stack(self, name: str, key: str) -> dict:  # pragma: no cover - must never run
            raise AssertionError("external stack must not be deleted")

    snapshot = SimpleNamespace(existed=False, restored_at=None)
    with pytest.raises(RuntimeError, match="refusing_to_delete_non_api_managed_stack"):
        restore_runtime_snapshot(
            FakeDb(),
            SimpleNamespace(email="admin@example.com"),
            SimpleNamespace(id="target-5"),
            SimpleNamespace(stack_name="demo"),
            Client(),
            snapshot,
        )

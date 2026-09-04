import base64
import os

os.environ.setdefault("DOCKGE_MANAGER_DATABASE_URL", "sqlite:///./test-manager.db")
os.environ.setdefault("DOCKGE_MANAGER_JWT_SECRET", "test-secret-that-is-at-least-thirty-two-characters-long")
os.environ.setdefault("DOCKGE_MANAGER_FERNET_KEY", base64.urlsafe_b64encode(b"x" * 32).decode("ascii"))
os.environ.setdefault("DOCKGE_MANAGER_ADMIN_PASSWORD", "test-password")
os.environ.setdefault("DOCKGE_MANAGER_HEALTH_POLL_ENABLED", "false")
os.environ.setdefault("DOCKGE_MANAGER_ALLOW_HTTP_TARGETS", "true")

from fastapi.testclient import TestClient  # noqa: E402
from sqlalchemy import select  # noqa: E402

from app.db import session_scope  # noqa: E402
from app.main import app  # noqa: E402
from app.models import CredentialRef  # noqa: E402


def login(client: TestClient) -> dict[str, str]:
    response = client.post(
        "/api/v1/auth/login",
        json={"email": "admin@localhost", "password": "test-password"},
    )
    assert response.status_code == 200
    return {"Authorization": "Bearer " + response.json()["access_token"]}


def test_health_and_bootstrap() -> None:
    with TestClient(app) as client:
        response = client.get("/api/health")
        assert response.status_code == 200
        assert response.json()["service"] == "dockge-manager"
        headers = login(client)
        assert client.get("/api/v1/workspaces", headers=headers).json()[0]["name"] == "Default"
        assert client.get("/api/v1/environments", headers=headers).json()[0]["name"] == "Default"


def test_target_secret_is_not_returned_or_stored_plaintext() -> None:
    secret = "0123456789abcdef0123456789abcdef"
    with TestClient(app) as client:
        headers = login(client)
        response = client.post(
            "/api/v1/targets",
            headers=headers,
            json={
                "name": "test-target",
                "base_url": "http://127.0.0.1:59999",
                "token": secret,
                "verify_tls": False,
            },
        )
        assert response.status_code == 201
        assert "token" not in response.json()

    with session_scope() as db:
        credential = db.scalar(select(CredentialRef).where(CredentialRef.name.like("dockge:test-target:%")))
        assert credential is not None
        assert secret not in credential.secret_ciphertext


def test_delete_target_returns_empty_204() -> None:
    with TestClient(app) as client:
        headers = login(client)
        created = client.post(
            "/api/v1/targets",
            headers=headers,
            json={
                "name": "delete-target",
                "base_url": "http://127.0.0.1:59998",
                "token": "abcdef0123456789abcdef0123456789",
                "verify_tls": False,
            },
        )
        assert created.status_code == 201
        deleted = client.delete(f"/api/v1/targets/{created.json()['id']}", headers=headers)
        assert deleted.status_code == 204
        assert deleted.content == b""

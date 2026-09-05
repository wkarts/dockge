import base64
import os

os.environ.setdefault("DOCKGE_MANAGER_DATABASE_URL", "sqlite:///./test-manager.db")
os.environ.setdefault("DOCKGE_MANAGER_JWT_SECRET", "test-secret-that-is-at-least-thirty-two-characters-long")
os.environ.setdefault("DOCKGE_MANAGER_FERNET_KEY", base64.urlsafe_b64encode(b"x" * 32).decode("ascii"))
os.environ.setdefault("DOCKGE_MANAGER_ADMIN_PASSWORD", "test-password")
os.environ.setdefault("DOCKGE_MANAGER_HEALTH_POLL_ENABLED", "false")

from app.deployment_engine import StackVerificationError, assess_stack_ps, verify_stack  # noqa: E402


def test_assess_stack_ps_requires_running_and_healthy() -> None:
    ok, reason = assess_stack_ps({"containers": [{"Name": "api", "State": "running", "Health": "healthy"}]})
    assert ok is True
    assert reason == "healthy"

    ok, reason = assess_stack_ps({"containers": [{"Name": "api", "State": "exited", "Health": ""}]})
    assert ok is False
    assert reason == "api_state_exited"

    ok, reason = assess_stack_ps({"containers": [{"Name": "api", "State": "running", "Health": "unhealthy"}]})
    assert ok is False
    assert reason == "api_health_unhealthy"


def test_verify_stack_retries_starting_health() -> None:
    class Client:
        calls = 0

        def ps(self, name: str) -> dict:
            assert name == "demo"
            self.calls += 1
            health = "starting" if self.calls == 1 else "healthy"
            return {"containers": [{"Name": "demo-api", "State": "running", "Health": health}]}

    client = Client()
    result = verify_stack(client, "demo", attempts=2, interval_seconds=0)
    assert client.calls == 2
    assert result["containers"][0]["Health"] == "healthy"


def test_verify_stack_fails_closed_when_no_container_is_healthy() -> None:
    class Client:
        def ps(self, name: str) -> dict:
            return {"containers": []}

    try:
        verify_stack(Client(), "demo", attempts=2, interval_seconds=0)
    except StackVerificationError as exc:
        assert exc.reason == "stack_has_no_containers"
    else:
        raise AssertionError("verification should fail when the stack has no running containers")

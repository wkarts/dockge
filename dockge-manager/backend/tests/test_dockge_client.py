from app.dockge_client import runtime_health, sanitize_stack_payload


def test_stack_payload_redacts_environment_secrets() -> None:
    original = {
        "stacks": [
            {
                "name": "app",
                "composeYAML": "services: {}",
                "composeENV": "SECRET=value",
                "compose_env": "OTHER=value",
            }
        ]
    }
    clean = sanitize_stack_payload(original)
    assert clean["stacks"][0]["composeYAML"] == "services: {}"
    assert "composeENV" not in clean["stacks"][0]
    assert "compose_env" not in clean["stacks"][0]
    assert original["stacks"][0]["composeENV"] == "SECRET=value"


def test_runtime_health_accepts_running_container_without_healthcheck() -> None:
    ok, detail = runtime_health({"containers": [{"Name": "web", "State": "running", "Health": ""}]})
    assert ok is True
    assert detail["containers"] == 1
    assert detail["failures"] == []


def test_runtime_health_rejects_starting_or_unhealthy_container() -> None:
    ok, detail = runtime_health(
        {
            "containers": [
                {"Name": "web", "State": "running", "Health": "starting"},
                {"Name": "worker", "State": "exited", "Health": ""},
            ]
        }
    )
    assert ok is False
    assert len(detail["failures"]) == 2


def test_runtime_health_rejects_empty_runtime() -> None:
    ok, detail = runtime_health({"containers": []})
    assert ok is False
    assert detail["error"] == "no_running_containers"

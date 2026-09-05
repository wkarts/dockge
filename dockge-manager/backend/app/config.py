from functools import lru_cache
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict


ROOT = Path(__file__).resolve().parents[2]


def _read_version() -> str:
    try:
        return (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    except OSError:
        return "0.0.0-dev"


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="DOCKGE_MANAGER_",
        case_sensitive=False,
        extra="ignore",
    )

    app_name: str = "Dockge Manager"
    version: str = _read_version()
    database_url: str = "sqlite:///./dockge-manager.db"
    jwt_secret: str = ""
    jwt_algorithm: str = "HS256"
    jwt_expire_minutes: int = 480
    fernet_key: str = ""
    admin_email: str = "admin@localhost"
    admin_password: str = ""
    allow_http_targets: bool = False
    health_poll_enabled: bool = True
    health_poll_seconds: int = 60
    deployment_verify_attempts: int = 8
    deployment_verify_interval_seconds: float = 2.0
    mutation_retry_attempts: int = 2
    mutation_retry_delay_seconds: float = 0.35
    static_dir: str = "/app/static"

    def validate_runtime_secrets(self) -> None:
        missing: list[str] = []
        if len(self.jwt_secret) < 32:
            missing.append("DOCKGE_MANAGER_JWT_SECRET (>= 32 chars)")
        if not self.fernet_key:
            missing.append("DOCKGE_MANAGER_FERNET_KEY")
        if not self.admin_password:
            missing.append("DOCKGE_MANAGER_ADMIN_PASSWORD")
        if missing:
            raise RuntimeError("Required Manager secrets are missing: " + ", ".join(missing))
        if self.deployment_verify_attempts < 1 or self.deployment_verify_attempts > 60:
            raise RuntimeError("DOCKGE_MANAGER_DEPLOYMENT_VERIFY_ATTEMPTS must be between 1 and 60")
        if self.deployment_verify_interval_seconds < 0 or self.deployment_verify_interval_seconds > 30:
            raise RuntimeError("DOCKGE_MANAGER_DEPLOYMENT_VERIFY_INTERVAL_SECONDS must be between 0 and 30")
        if self.mutation_retry_attempts < 0 or self.mutation_retry_attempts > 5:
            raise RuntimeError("DOCKGE_MANAGER_MUTATION_RETRY_ATTEMPTS must be between 0 and 5")
        if self.mutation_retry_delay_seconds < 0 or self.mutation_retry_delay_seconds > 10:
            raise RuntimeError("DOCKGE_MANAGER_MUTATION_RETRY_DELAY_SECONDS must be between 0 and 10")


@lru_cache
def get_settings() -> Settings:
    return Settings()

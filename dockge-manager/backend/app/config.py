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


@lru_cache
def get_settings() -> Settings:
    return Settings()

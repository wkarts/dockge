import asyncio
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from sqlalchemy import select

from .api import router
from .config import get_settings
from .db import Base, engine, session_scope
from .models import Environment, User, Workspace
from .monitoring import monitoring_loop
from .security import SecretBox, hash_password


settings = get_settings()


def bootstrap() -> None:
    settings.validate_runtime_secrets()
    SecretBox(settings.fernet_key)
    Base.metadata.create_all(bind=engine)
    with session_scope() as db:
        workspace = db.scalar(select(Workspace).order_by(Workspace.created_at).limit(1))
        if workspace is None:
            workspace = Workspace(name="Default")
            db.add(workspace)
            db.flush()
        environment = db.scalar(select(Environment).order_by(Environment.created_at).limit(1))
        if environment is None:
            db.add(Environment(workspace_id=workspace.id, name="Default"))
        user = db.scalar(select(User).where(User.email == settings.admin_email.lower()))
        if user is None:
            db.add(
                User(
                    email=settings.admin_email.lower(),
                    password_hash=hash_password(settings.admin_password),
                    is_active=True,
                    is_admin=True,
                )
            )


@asynccontextmanager
async def lifespan(_: FastAPI):
    bootstrap()
    stop = asyncio.Event()
    task: asyncio.Task | None = None
    if settings.health_poll_enabled:
        task = asyncio.create_task(monitoring_loop(stop))
    try:
        yield
    finally:
        stop.set()
        if task is not None:
            await task


app = FastAPI(
    title="Dockge Manager API",
    version=settings.version,
    docs_url="/api/docs",
    openapi_url="/api/openapi.json",
    lifespan=lifespan,
)


@app.middleware("http")
async def security_headers(request, call_next):
    response = await call_next(request)
    response.headers["X-Content-Type-Options"] = "nosniff"
    response.headers["X-Frame-Options"] = "DENY"
    response.headers["Referrer-Policy"] = "no-referrer"
    response.headers["Permissions-Policy"] = "camera=(), microphone=(), geolocation=()"
    response.headers["Content-Security-Policy"] = (
        "default-src 'self'; "
        "base-uri 'self'; frame-ancestors 'none'; object-src 'none'; "
        "script-src 'self'; style-src 'self' 'unsafe-inline'; "
        "img-src 'self' data:; connect-src 'self'"
    )
    return response


@app.get("/api/health")
def health() -> dict:
    return {"ok": True, "service": "dockge-manager", "version": settings.version}


app.include_router(router)

static_dir = Path(settings.static_dir)
if static_dir.exists():
    app.mount("/", StaticFiles(directory=static_dir, html=True), name="frontend")

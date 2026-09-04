from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field, HttpUrl


class LoginRequest(BaseModel):
    email: str
    password: str


class LoginResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"


class UserOut(BaseModel):
    id: str
    email: str
    is_admin: bool

    model_config = ConfigDict(from_attributes=True)


class WorkspaceCreate(BaseModel):
    name: str = Field(min_length=1, max_length=120)


class WorkspaceOut(BaseModel):
    id: str
    name: str
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)


class EnvironmentCreate(BaseModel):
    workspace_id: str
    name: str = Field(min_length=1, max_length=120)


class EnvironmentOut(BaseModel):
    id: str
    workspace_id: str
    name: str
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)


class TargetCreate(BaseModel):
    environment_id: str | None = None
    name: str = Field(min_length=1, max_length=120)
    base_url: HttpUrl
    token: str = Field(min_length=16)
    verify_tls: bool = True
    enabled: bool = True


class TargetUpdate(BaseModel):
    environment_id: str | None = None
    name: str | None = Field(default=None, min_length=1, max_length=120)
    base_url: HttpUrl | None = None
    token: str | None = Field(default=None, min_length=16)
    verify_tls: bool | None = None
    enabled: bool | None = None


class TargetOut(BaseModel):
    id: str
    environment_id: str
    name: str
    base_url: str
    verify_tls: bool
    enabled: bool
    last_seen_at: datetime | None
    last_version: str | None
    created_at: datetime
    updated_at: datetime

    model_config = ConfigDict(from_attributes=True)


class StackApplyRequest(BaseModel):
    compose_yaml: str = Field(min_length=1)
    compose_env: str | None = None
    owner: str | None = None
    adopt: bool = False


class ApplicationCreate(BaseModel):
    name: str = Field(min_length=1, max_length=160)
    description: str = ""


class ApplicationOut(BaseModel):
    id: str
    name: str
    description: str
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)


class DeploymentCreate(BaseModel):
    application_id: str
    target_id: str
    stack_name: str = Field(pattern=r"^[a-z0-9][a-z0-9_-]{0,127}$")
    compose_yaml: str = Field(min_length=1)
    compose_env: str = ""
    adopt_external: bool = False


class RevisionCreate(BaseModel):
    compose_yaml: str = Field(min_length=1)
    compose_env: str = ""
    adopt_external: bool = False


class RevisionOut(BaseModel):
    id: str
    revision: int
    compose_yaml: str
    adopt_external: bool
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)


class DeploymentOut(BaseModel):
    id: str
    application_id: str
    target_id: str
    stack_name: str
    status: str
    current_revision: int
    created_at: datetime
    updated_at: datetime

    model_config = ConfigDict(from_attributes=True)


class OperationOut(BaseModel):
    id: str
    target_id: str
    stack_name: str
    action: str
    idempotency_key: str
    status: str
    http_status: int | None
    response_json: dict
    created_at: datetime
    completed_at: datetime | None

    model_config = ConfigDict(from_attributes=True)


class AuditEventOut(BaseModel):
    id: str
    actor: str
    event_type: str
    target_id: str | None
    resource: str
    payload: dict
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)


class HealthSnapshotOut(BaseModel):
    id: str
    target_id: str
    ok: bool
    version: str | None
    detail_json: dict
    created_at: datetime

    model_config = ConfigDict(from_attributes=True)

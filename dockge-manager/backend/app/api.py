from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

from fastapi import APIRouter, HTTPException, Query, status
from sqlalchemy import func, select
from sqlalchemy.exc import IntegrityError

from .config import get_settings
from .dependencies import CurrentUser, Db
from .deployment_engine import StackVerificationError, verify_stack
from .dockge_client import DockgeRequestError, validate_target_url
from .models import (
    Application,
    AuditEvent,
    CredentialRef,
    Deployment,
    DeploymentRevision,
    DeploymentSnapshot,
    DockgeTarget,
    Environment,
    HealthSnapshot,
    Operation,
    User,
    Workspace,
)
from .schemas import (
    ApplicationCreate,
    ApplicationOut,
    AuditEventOut,
    DeploymentCreate,
    DeploymentOut,
    DeploymentSnapshotOut,
    EnvironmentCreate,
    EnvironmentOut,
    HealthSnapshotOut,
    LoginRequest,
    LoginResponse,
    OperationOut,
    RevisionCreate,
    RevisionOut,
    StackApplyRequest,
    TargetCreate,
    TargetOut,
    TargetUpdate,
    UserOut,
    WorkspaceCreate,
    WorkspaceOut,
)
from .security import SecretBox, create_access_token, verify_password
from .service import audit, client_for, run_mutation


router = APIRouter(prefix="/api/v1")


def secret_box() -> SecretBox:
    return SecretBox(get_settings().fernet_key)


def target_or_404(db: Db, target_id: str) -> DockgeTarget:
    target = db.get(DockgeTarget, target_id)
    if target is None:
        raise HTTPException(status_code=404, detail="target_not_found")
    return target


def deployment_or_404(db: Db, deployment_id: str) -> Deployment:
    deployment = db.get(Deployment, deployment_id)
    if deployment is None:
        raise HTTPException(status_code=404, detail="deployment_not_found")
    return deployment


def environment_or_404(db: Db, environment_id: str) -> Environment:
    environment = db.get(Environment, environment_id)
    if environment is None:
        raise HTTPException(status_code=404, detail="environment_not_found")
    return environment


def default_environment(db: Db) -> Environment:
    environment = db.scalar(select(Environment).order_by(Environment.created_at).limit(1))
    if environment is None:
        raise HTTPException(status_code=409, detail="no_environment_configured")
    return environment


def translate_dockge_error(exc: DockgeRequestError) -> HTTPException:
    status_code = exc.status_code if 400 <= exc.status_code < 600 else 502
    return HTTPException(status_code=status_code, detail=exc.detail)


def runtime_snapshot(db: Db, deployment: Deployment, client, revision: int, reason: str) -> DeploymentSnapshot:
    existed = True
    stack: dict = {}
    try:
        stack = client.stack(deployment.stack_name)
    except DockgeRequestError as exc:
        if exc.status_code == 404:
            existed = False
        else:
            raise

    compose_yaml = str(stack.get("composeYAML") or "") if existed else ""
    compose_env = str(stack.get("composeENV") or "") if existed else ""
    snapshot = DeploymentSnapshot(
        deployment_id=deployment.id,
        captured_for_revision=revision,
        existed=existed,
        api_managed=bool(stack.get("api_managed", False)) if existed else False,
        compose_yaml=compose_yaml,
        compose_env_ciphertext=secret_box().encrypt(compose_env),
        reason=reason,
    )
    db.add(snapshot)
    db.flush()
    return snapshot


def restore_runtime_snapshot(
    db: Db,
    user: CurrentUser,
    target: DockgeTarget,
    deployment: Deployment,
    client,
    snapshot: DeploymentSnapshot,
) -> dict:
    if snapshot.existed:
        payload = {
            "compose_yaml": snapshot.compose_yaml,
            "compose_env": secret_box().decrypt(snapshot.compose_env_ciphertext),
            "owner": user.email,
            # If an external stack was explicitly adopted by the failed
            # deployment, the marker now exists. Restoring through the API must
            # therefore explicitly acknowledge that adoption. The previous
            # Compose/.env are restored, while the explicit adoption remains.
            "adopt": not snapshot.api_managed,
        }
        run_mutation(
            db,
            target,
            user.email,
            deployment.stack_name,
            "auto-rollback.apply",
            lambda key: client.apply_stack(deployment.stack_name, payload, key),
        )
        run_mutation(
            db,
            target,
            user.email,
            deployment.stack_name,
            "auto-rollback.up",
            lambda key: client.action(deployment.stack_name, "up", key),
        )
        settings = get_settings()
        ps = verify_stack(
            client,
            deployment.stack_name,
            settings.deployment_verify_attempts,
            settings.deployment_verify_interval_seconds,
        )
        snapshot.restored_at = datetime.now(timezone.utc)
        return {"restored": "previous_runtime", "ps": ps, "adoption_persisted": not snapshot.api_managed}

    # The failed deployment created a stack that did not exist before. Delete
    # only that API-managed stack. Dockge's DELETE intentionally does not remove
    # named volumes.
    result = run_mutation(
        db,
        target,
        user.email,
        deployment.stack_name,
        "auto-rollback.delete-created-stack",
        lambda key: client.delete_stack(deployment.stack_name, key),
    )
    snapshot.restored_at = datetime.now(timezone.utc)
    return {"restored": "stack_absence", "delete": result}


@router.post("/auth/login", response_model=LoginResponse)
def login(body: LoginRequest, db: Db) -> LoginResponse:
    user = db.scalar(select(User).where(func.lower(User.email) == body.email.lower()))
    if user is None or not user.is_active or not verify_password(body.password, user.password_hash):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="invalid_credentials")
    return LoginResponse(access_token=create_access_token(user.id, user.email))


@router.get("/auth/me", response_model=UserOut)
def me(user: CurrentUser) -> User:
    return user


@router.get("/workspaces", response_model=list[WorkspaceOut])
def list_workspaces(db: Db, user: CurrentUser) -> list[Workspace]:
    del user
    return list(db.scalars(select(Workspace).order_by(Workspace.name)))


@router.post("/workspaces", response_model=WorkspaceOut, status_code=201)
def create_workspace(body: WorkspaceCreate, db: Db, user: CurrentUser) -> Workspace:
    workspace = Workspace(name=body.name.strip())
    db.add(workspace)
    audit(db, user.email, "workspace.create", resource=workspace.name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="workspace_name_already_exists") from exc
    db.refresh(workspace)
    return workspace


@router.get("/environments", response_model=list[EnvironmentOut])
def list_environments(db: Db, user: CurrentUser) -> list[Environment]:
    del user
    return list(db.scalars(select(Environment).order_by(Environment.name)))


@router.post("/environments", response_model=EnvironmentOut, status_code=201)
def create_environment(body: EnvironmentCreate, db: Db, user: CurrentUser) -> Environment:
    if db.get(Workspace, body.workspace_id) is None:
        raise HTTPException(status_code=404, detail="workspace_not_found")
    environment = Environment(workspace_id=body.workspace_id, name=body.name.strip())
    db.add(environment)
    audit(db, user.email, "environment.create", resource=environment.name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="environment_name_already_exists_in_workspace") from exc
    db.refresh(environment)
    return environment


@router.get("/targets", response_model=list[TargetOut])
def list_targets(db: Db, user: CurrentUser) -> list[DockgeTarget]:
    del user
    return list(db.scalars(select(DockgeTarget).order_by(DockgeTarget.name)))


@router.post("/targets", response_model=TargetOut, status_code=201)
def create_target(body: TargetCreate, db: Db, user: CurrentUser) -> DockgeTarget:
    try:
        base_url = validate_target_url(str(body.base_url))
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    environment = environment_or_404(db, body.environment_id) if body.environment_id else default_environment(db)
    credential = CredentialRef(
        name=f"dockge:{body.name.strip()}:{uuid4().hex[:12]}",
        kind="dockge_bearer",
        secret_ciphertext=secret_box().encrypt(body.token),
    )
    db.add(credential)
    db.flush()
    target = DockgeTarget(
        environment_id=environment.id,
        credential_id=credential.id,
        name=body.name.strip(),
        base_url=base_url,
        verify_tls=body.verify_tls,
        enabled=body.enabled,
    )
    db.add(target)
    db.flush()
    audit(db, user.email, "target.create", target_id=target.id, resource=target.name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="target_name_already_exists") from exc
    db.refresh(target)
    return target


@router.patch("/targets/{target_id}", response_model=TargetOut)
def update_target(target_id: str, body: TargetUpdate, db: Db, user: CurrentUser) -> DockgeTarget:
    target = target_or_404(db, target_id)
    patch = body.model_dump(exclude_unset=True)
    if "environment_id" in patch and patch["environment_id"] is not None:
        environment_or_404(db, patch["environment_id"])
    elif patch.get("environment_id") is None:
        patch.pop("environment_id", None)
    if "base_url" in patch:
        try:
            patch["base_url"] = validate_target_url(str(patch["base_url"]))
        except ValueError as exc:
            raise HTTPException(status_code=400, detail=str(exc)) from exc
    if "token" in patch:
        target.credential.secret_ciphertext = secret_box().encrypt(patch.pop("token"))
        target.credential.rotated_at = datetime.now(timezone.utc)
    for field, value in patch.items():
        setattr(target, field, value)
    audit(db, user.email, "target.update", target_id=target.id, resource=target.name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="target_update_conflict") from exc
    db.refresh(target)
    return target


@router.delete("/targets/{target_id}", status_code=204, response_model=None)
def delete_target(target_id: str, db: Db, user: CurrentUser) -> None:
    target = target_or_404(db, target_id)
    credential = target.credential
    audit(db, user.email, "target.delete", target_id=target.id, resource=target.name)
    db.delete(target)
    db.flush()
    db.delete(credential)
    db.commit()


@router.post("/targets/{target_id}/test")
def test_target(target_id: str, db: Db, user: CurrentUser) -> dict:
    target = target_or_404(db, target_id)
    try:
        data = client_for(target, secret_box()).health()
    except DockgeRequestError as exc:
        db.add(HealthSnapshot(target_id=target.id, ok=False, version=target.last_version, detail_json={"status": exc.status_code}))
        audit(db, user.email, "target.test.failed", target_id=target.id, payload={"status": exc.status_code})
        db.commit()
        raise translate_dockge_error(exc) from exc
    target.last_seen_at = datetime.now(timezone.utc)
    target.last_version = str(data.get("version") or "") or None
    db.add(HealthSnapshot(target_id=target.id, ok=bool(data.get("ok", True)), version=target.last_version, detail_json=data))
    audit(db, user.email, "target.test.succeeded", target_id=target.id)
    db.commit()
    return data


@router.get("/targets/{target_id}/info")
def target_info(target_id: str, db: Db, user: CurrentUser) -> dict:
    target = target_or_404(db, target_id)
    try:
        data = client_for(target, secret_box()).info()
    except DockgeRequestError as exc:
        raise translate_dockge_error(exc) from exc
    target.last_seen_at = datetime.now(timezone.utc)
    target.last_version = str(data.get("version") or "") or target.last_version
    audit(db, user.email, "target.info", target_id=target.id)
    db.commit()
    return data


@router.get("/targets/{target_id}/stacks")
def target_stacks(target_id: str, db: Db, user: CurrentUser) -> dict:
    target = target_or_404(db, target_id)
    try:
        data = client_for(target, secret_box()).stacks()
    except DockgeRequestError as exc:
        raise translate_dockge_error(exc) from exc
    audit(db, user.email, "stack.list", target_id=target.id)
    db.commit()
    return data


@router.get("/targets/{target_id}/stacks/{stack_name}")
def target_stack(target_id: str, stack_name: str, db: Db, user: CurrentUser) -> dict:
    target = target_or_404(db, target_id)
    try:
        data = client_for(target, secret_box()).stack(stack_name)
    except DockgeRequestError as exc:
        raise translate_dockge_error(exc) from exc
    audit(db, user.email, "stack.read", target_id=target.id, resource=stack_name)
    db.commit()
    return data


@router.put("/targets/{target_id}/stacks/{stack_name}")
def apply_stack(target_id: str, stack_name: str, body: StackApplyRequest, db: Db, user: CurrentUser) -> dict:
    target = target_or_404(db, target_id)
    client = client_for(target, secret_box())
    payload = {
        "compose_yaml": body.compose_yaml,
        "owner": body.owner or user.email,
        "adopt": body.adopt,
    }
    if body.compose_env is not None:
        payload["compose_env"] = body.compose_env
    try:
        result = run_mutation(
            db,
            target,
            user.email,
            stack_name,
            "apply",
            lambda key: client.apply_stack(stack_name, payload, key),
        )
        db.commit()
        return result
    except DockgeRequestError as exc:
        db.commit()
        raise translate_dockge_error(exc) from exc


@router.post("/targets/{target_id}/stacks/{stack_name}/actions/{action}")
def stack_action(target_id: str, stack_name: str, action: str, db: Db, user: CurrentUser) -> dict:
    supported = {"pull", "up", "down", "start", "stop", "restart"}
    if action not in supported:
        raise HTTPException(status_code=400, detail={"error": "unsupported_action", "supported": sorted(supported)})
    target = target_or_404(db, target_id)
    client = client_for(target, secret_box())
    try:
        result = run_mutation(
            db,
            target,
            user.email,
            stack_name,
            action,
            lambda key: client.action(stack_name, action, key),
        )
        db.commit()
        return result
    except DockgeRequestError as exc:
        db.commit()
        raise translate_dockge_error(exc) from exc


@router.get("/targets/{target_id}/stacks/{stack_name}/logs")
def stack_logs(
    target_id: str,
    stack_name: str,
    db: Db,
    user: CurrentUser,
    tail: int = Query(default=200, ge=1, le=2000),
) -> dict:
    target = target_or_404(db, target_id)
    try:
        data = client_for(target, secret_box()).logs(stack_name, tail)
    except DockgeRequestError as exc:
        raise translate_dockge_error(exc) from exc
    audit(db, user.email, "stack.logs", target_id=target.id, resource=stack_name, payload={"tail": tail})
    db.commit()
    return data


@router.get("/applications", response_model=list[ApplicationOut])
def list_applications(db: Db, user: CurrentUser) -> list[Application]:
    del user
    return list(db.scalars(select(Application).order_by(Application.name)))


@router.post("/applications", response_model=ApplicationOut, status_code=201)
def create_application(body: ApplicationCreate, db: Db, user: CurrentUser) -> Application:
    app = Application(name=body.name.strip(), description=body.description)
    db.add(app)
    audit(db, user.email, "application.create", resource=app.name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="application_name_already_exists") from exc
    db.refresh(app)
    return app


@router.get("/deployments", response_model=list[DeploymentOut])
def list_deployments(db: Db, user: CurrentUser) -> list[Deployment]:
    del user
    return list(db.scalars(select(Deployment).order_by(Deployment.updated_at.desc())))


@router.post("/deployments", response_model=DeploymentOut, status_code=201)
def create_deployment(body: DeploymentCreate, db: Db, user: CurrentUser) -> Deployment:
    if db.get(Application, body.application_id) is None:
        raise HTTPException(status_code=404, detail="application_not_found")
    target_or_404(db, body.target_id)
    deployment = Deployment(
        application_id=body.application_id,
        target_id=body.target_id,
        stack_name=body.stack_name,
        status="DRAFT",
        current_revision=1,
        active_revision=0,
    )
    db.add(deployment)
    db.flush()
    db.add(
        DeploymentRevision(
            deployment_id=deployment.id,
            revision=1,
            compose_yaml=body.compose_yaml,
            compose_env_ciphertext=secret_box().encrypt(body.compose_env),
            adopt_external=body.adopt_external,
        )
    )
    audit(db, user.email, "deployment.create", target_id=body.target_id, resource=body.stack_name)
    try:
        db.commit()
    except IntegrityError as exc:
        db.rollback()
        raise HTTPException(status_code=409, detail="deployment_for_target_stack_already_exists") from exc
    db.refresh(deployment)
    return deployment


@router.get("/deployments/{deployment_id}/revisions", response_model=list[RevisionOut])
def list_revisions(deployment_id: str, db: Db, user: CurrentUser) -> list[DeploymentRevision]:
    del user
    deployment_or_404(db, deployment_id)
    return list(
        db.scalars(
            select(DeploymentRevision)
            .where(DeploymentRevision.deployment_id == deployment_id)
            .order_by(DeploymentRevision.revision.desc())
        )
    )


@router.post("/deployments/{deployment_id}/revisions", response_model=RevisionOut, status_code=201)
def create_revision(deployment_id: str, body: RevisionCreate, db: Db, user: CurrentUser) -> DeploymentRevision:
    deployment = deployment_or_404(db, deployment_id)
    next_revision = (db.scalar(select(func.max(DeploymentRevision.revision)).where(DeploymentRevision.deployment_id == deployment.id)) or 0) + 1
    revision = DeploymentRevision(
        deployment_id=deployment.id,
        revision=next_revision,
        compose_yaml=body.compose_yaml,
        compose_env_ciphertext=secret_box().encrypt(body.compose_env),
        adopt_external=body.adopt_external,
    )
    db.add(revision)
    deployment.current_revision = next_revision
    deployment.status = "DRAFT"
    deployment.last_error = ""
    audit(db, user.email, "deployment.revision.create", target_id=deployment.target_id, resource=deployment.stack_name, payload={"revision": next_revision})
    db.commit()
    db.refresh(revision)
    return revision


@router.get("/deployments/{deployment_id}/snapshots", response_model=list[DeploymentSnapshotOut])
def list_deployment_snapshots(deployment_id: str, db: Db, user: CurrentUser) -> list[DeploymentSnapshot]:
    del user
    deployment_or_404(db, deployment_id)
    return list(
        db.scalars(
            select(DeploymentSnapshot)
            .where(DeploymentSnapshot.deployment_id == deployment_id)
            .order_by(DeploymentSnapshot.created_at.desc())
        )
    )


def execute_revision(db: Db, user: CurrentUser, deployment: Deployment, revision: DeploymentRevision, rollback: bool = False) -> dict:
    target = target_or_404(db, deployment.target_id)
    client = client_for(target, secret_box())
    settings = get_settings()
    previous_active = deployment.active_revision
    previous_current = deployment.current_revision
    snapshot = runtime_snapshot(
        db,
        deployment,
        client,
        revision.revision,
        "pre-rollback" if rollback else "pre-deploy",
    )
    deployment.status = "ROLLING_BACK" if rollback else "DEPLOYING"
    deployment.last_error = ""
    db.flush()

    apply_payload = {
        "compose_yaml": revision.compose_yaml,
        "compose_env": secret_box().decrypt(revision.compose_env_ciphertext),
        "owner": user.email,
        "adopt": revision.adopt_external,
    }
    prefix = "rollback" if rollback else "deploy"
    mutated = False
    try:
        run_mutation(
            db,
            target,
            user.email,
            deployment.stack_name,
            f"{prefix}.apply",
            lambda key: client.apply_stack(deployment.stack_name, apply_payload, key),
        )
        mutated = True
        run_mutation(
            db,
            target,
            user.email,
            deployment.stack_name,
            f"{prefix}.up",
            lambda key: client.action(deployment.stack_name, "up", key),
        )
        deployment.status = "VERIFYING"
        db.flush()
        ps = verify_stack(
            client,
            deployment.stack_name,
            settings.deployment_verify_attempts,
            settings.deployment_verify_interval_seconds,
        )
        deployment.current_revision = revision.revision
        deployment.active_revision = revision.revision
        deployment.status = "ROLLED_BACK" if rollback else "HEALTHY"
        deployment.last_error = ""
        deployment.last_deployed_at = datetime.now(timezone.utc)
        audit(
            db,
            user.email,
            f"deployment.{prefix}.succeeded",
            target_id=target.id,
            resource=deployment.stack_name,
            payload={"revision": revision.revision, "snapshot_id": snapshot.id},
        )
        db.commit()
        return {
            "ok": True,
            "deployment_id": deployment.id,
            "revision": revision.revision,
            "active_revision": deployment.active_revision,
            "status": deployment.status,
            "ps": ps,
        }
    except (DockgeRequestError, StackVerificationError) as exc:
        if isinstance(exc, DockgeRequestError):
            original_status = exc.status_code
            original_detail = exc.detail
        else:
            original_status = 502
            original_detail = {"error": "deployment_verification_failed", "reason": exc.reason, "ps": exc.payload}

        deployment.last_error = str(original_detail)[:4000]
        rollback_result: dict | None = None
        rollback_error: str | None = None
        if mutated:
            try:
                deployment.status = "ROLLING_BACK"
                db.flush()
                rollback_result = restore_runtime_snapshot(db, user, target, deployment, client, snapshot)
                deployment.status = "ROLLED_BACK"
                deployment.active_revision = previous_active
                deployment.current_revision = previous_current
            except (DockgeRequestError, StackVerificationError, RuntimeError) as rollback_exc:
                rollback_error = str(rollback_exc)
                deployment.status = "ROLLBACK_FAILED"
        else:
            # A precondition/API failure before the first successful mutation
            # must not adopt, delete or otherwise alter the target stack.
            deployment.status = "FAILED"

        audit(
            db,
            user.email,
            f"deployment.{prefix}.failed",
            target_id=target.id,
            resource=deployment.stack_name,
            payload={
                "revision": revision.revision,
                "status": original_status,
                "snapshot_id": snapshot.id,
                "mutated": mutated,
                "rollback": rollback_result,
                "rollback_error": rollback_error,
            },
        )
        db.commit()

        if rollback_result is not None:
            raise HTTPException(
                status_code=502,
                detail={
                    "error": "deployment_failed_and_runtime_was_restored",
                    "cause": original_detail,
                    "status": deployment.status,
                    "rollback": rollback_result,
                },
            ) from exc
        if rollback_error is not None:
            raise HTTPException(
                status_code=502,
                detail={
                    "error": "deployment_failed_and_rollback_failed",
                    "cause": original_detail,
                    "rollback_error": rollback_error,
                    "status": deployment.status,
                },
            ) from exc
        if isinstance(exc, DockgeRequestError):
            raise translate_dockge_error(exc) from exc
        raise HTTPException(status_code=502, detail=original_detail) from exc


@router.post("/deployments/{deployment_id}/deploy")
def deploy_current(deployment_id: str, db: Db, user: CurrentUser) -> dict:
    deployment = deployment_or_404(db, deployment_id)
    revision = db.scalar(
        select(DeploymentRevision).where(
            DeploymentRevision.deployment_id == deployment.id,
            DeploymentRevision.revision == deployment.current_revision,
        )
    )
    if revision is None:
        raise HTTPException(status_code=409, detail="deployment_revision_not_found")
    return execute_revision(db, user, deployment, revision)


@router.post("/deployments/{deployment_id}/rollback")
def rollback_deployment(deployment_id: str, db: Db, user: CurrentUser) -> dict:
    deployment = deployment_or_404(db, deployment_id)
    if deployment.active_revision <= 1:
        raise HTTPException(status_code=409, detail="no_previous_active_revision_available")
    revision = db.scalar(
        select(DeploymentRevision)
        .where(
            DeploymentRevision.deployment_id == deployment.id,
            DeploymentRevision.revision < deployment.active_revision,
        )
        .order_by(DeploymentRevision.revision.desc())
        .limit(1)
    )
    if revision is None:
        raise HTTPException(status_code=409, detail="no_previous_revision_available")
    return execute_revision(db, user, deployment, revision, rollback=True)


@router.get("/operations", response_model=list[OperationOut])
def operations(db: Db, user: CurrentUser, limit: int = Query(default=100, ge=1, le=500)) -> list[Operation]:
    del user
    return list(db.scalars(select(Operation).order_by(Operation.created_at.desc()).limit(limit)))


@router.get("/audit", response_model=list[AuditEventOut])
def audit_events(db: Db, user: CurrentUser, limit: int = Query(default=100, ge=1, le=500)) -> list[AuditEvent]:
    del user
    return list(db.scalars(select(AuditEvent).order_by(AuditEvent.created_at.desc()).limit(limit)))


@router.get("/health-snapshots", response_model=list[HealthSnapshotOut])
def health_snapshots(
    db: Db,
    user: CurrentUser,
    target_id: str | None = None,
    limit: int = Query(default=100, ge=1, le=500),
) -> list[HealthSnapshot]:
    del user
    statement = select(HealthSnapshot)
    if target_id:
        statement = statement.where(HealthSnapshot.target_id == target_id)
    return list(db.scalars(statement.order_by(HealthSnapshot.created_at.desc()).limit(limit)))

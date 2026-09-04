"""Initial Dockge Manager schema.

Revision ID: 0001_initial
Revises:
"""
from alembic import op
import sqlalchemy as sa


revision = "0001_initial"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "users",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("email", sa.String(320), nullable=False),
        sa.Column("password_hash", sa.String(255), nullable=False),
        sa.Column("is_active", sa.Boolean(), nullable=False),
        sa.Column("is_admin", sa.Boolean(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("email"),
    )
    op.create_index("ix_users_email", "users", ["email"], unique=False)

    op.create_table(
        "workspaces",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("name", sa.String(120), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("name"),
    )
    op.create_index("ix_workspaces_name", "workspaces", ["name"], unique=False)

    op.create_table(
        "environments",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("workspace_id", sa.String(36), sa.ForeignKey("workspaces.id", ondelete="CASCADE"), nullable=False),
        sa.Column("name", sa.String(120), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("workspace_id", "name", name="uq_environment_workspace_name"),
    )
    op.create_index("ix_environments_workspace_id", "environments", ["workspace_id"], unique=False)

    op.create_table(
        "credential_refs",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("name", sa.String(160), nullable=False),
        sa.Column("kind", sa.String(64), nullable=False),
        sa.Column("secret_ciphertext", sa.Text(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("rotated_at", sa.DateTime(timezone=True), nullable=True),
        sa.UniqueConstraint("name"),
    )
    op.create_index("ix_credential_refs_name", "credential_refs", ["name"], unique=False)

    op.create_table(
        "dockge_targets",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("environment_id", sa.String(36), sa.ForeignKey("environments.id", ondelete="RESTRICT"), nullable=False),
        sa.Column("credential_id", sa.String(36), sa.ForeignKey("credential_refs.id", ondelete="RESTRICT"), nullable=False),
        sa.Column("name", sa.String(120), nullable=False),
        sa.Column("base_url", sa.String(2048), nullable=False),
        sa.Column("verify_tls", sa.Boolean(), nullable=False),
        sa.Column("enabled", sa.Boolean(), nullable=False),
        sa.Column("last_seen_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("last_version", sa.String(64), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("credential_id"),
        sa.UniqueConstraint("name"),
    )
    op.create_index("ix_dockge_targets_environment_id", "dockge_targets", ["environment_id"], unique=False)
    op.create_index("ix_dockge_targets_credential_id", "dockge_targets", ["credential_id"], unique=False)
    op.create_index("ix_dockge_targets_name", "dockge_targets", ["name"], unique=False)

    op.create_table(
        "applications",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("name", sa.String(160), nullable=False),
        sa.Column("description", sa.Text(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("name"),
    )
    op.create_index("ix_applications_name", "applications", ["name"], unique=False)

    op.create_table(
        "deployments",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("application_id", sa.String(36), sa.ForeignKey("applications.id", ondelete="CASCADE"), nullable=False),
        sa.Column("target_id", sa.String(36), sa.ForeignKey("dockge_targets.id", ondelete="CASCADE"), nullable=False),
        sa.Column("stack_name", sa.String(128), nullable=False),
        sa.Column("status", sa.String(32), nullable=False),
        sa.Column("current_revision", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("target_id", "stack_name", name="uq_deployment_target_stack"),
    )
    op.create_index("ix_deployments_application_id", "deployments", ["application_id"], unique=False)
    op.create_index("ix_deployments_target_id", "deployments", ["target_id"], unique=False)
    op.create_index("ix_deployments_stack_name", "deployments", ["stack_name"], unique=False)

    op.create_table(
        "deployment_revisions",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("deployment_id", sa.String(36), sa.ForeignKey("deployments.id", ondelete="CASCADE"), nullable=False),
        sa.Column("revision", sa.Integer(), nullable=False),
        sa.Column("compose_yaml", sa.Text(), nullable=False),
        sa.Column("compose_env_ciphertext", sa.Text(), nullable=False),
        sa.Column("adopt_external", sa.Boolean(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("deployment_id", "revision", name="uq_deployment_revision"),
    )
    op.create_index("ix_deployment_revisions_deployment_id", "deployment_revisions", ["deployment_id"], unique=False)

    op.create_table(
        "operations",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("target_id", sa.String(36), sa.ForeignKey("dockge_targets.id", ondelete="CASCADE"), nullable=False),
        sa.Column("stack_name", sa.String(128), nullable=False),
        sa.Column("action", sa.String(64), nullable=False),
        sa.Column("idempotency_key", sa.String(128), nullable=False),
        sa.Column("status", sa.String(32), nullable=False),
        sa.Column("http_status", sa.Integer(), nullable=True),
        sa.Column("response_json", sa.JSON(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("completed_at", sa.DateTime(timezone=True), nullable=True),
        sa.UniqueConstraint("idempotency_key"),
    )
    op.create_index("ix_operations_target_id", "operations", ["target_id"], unique=False)
    op.create_index("ix_operations_idempotency_key", "operations", ["idempotency_key"], unique=False)

    op.create_table(
        "audit_events",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("actor", sa.String(320), nullable=False),
        sa.Column("event_type", sa.String(128), nullable=False),
        sa.Column("target_id", sa.String(36), nullable=True),
        sa.Column("resource", sa.String(256), nullable=False),
        sa.Column("payload", sa.JSON(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("ix_audit_events_event_type", "audit_events", ["event_type"], unique=False)
    op.create_index("ix_audit_events_target_id", "audit_events", ["target_id"], unique=False)
    op.create_index("ix_audit_events_created_at", "audit_events", ["created_at"], unique=False)

    op.create_table(
        "health_snapshots",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("target_id", sa.String(36), sa.ForeignKey("dockge_targets.id", ondelete="CASCADE"), nullable=False),
        sa.Column("ok", sa.Boolean(), nullable=False),
        sa.Column("version", sa.String(64), nullable=True),
        sa.Column("detail_json", sa.JSON(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("ix_health_snapshots_target_id", "health_snapshots", ["target_id"], unique=False)
    op.create_index("ix_health_snapshots_created_at", "health_snapshots", ["created_at"], unique=False)


def downgrade() -> None:
    op.drop_table("health_snapshots")
    op.drop_table("audit_events")
    op.drop_table("operations")
    op.drop_table("deployment_revisions")
    op.drop_table("deployments")
    op.drop_table("applications")
    op.drop_table("dockge_targets")
    op.drop_table("credential_refs")
    op.drop_table("environments")
    op.drop_table("workspaces")
    op.drop_table("users")

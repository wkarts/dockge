"""Deployment runtime snapshots and active revision.

Revision ID: 0002_deployment_safety
Revises: 0001_initial
"""
from alembic import op
import sqlalchemy as sa


revision = "0002_deployment_safety"
down_revision = "0001_initial"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("deployments", sa.Column("active_revision", sa.Integer(), nullable=False, server_default="0"))
    op.add_column("deployments", sa.Column("last_error", sa.Text(), nullable=False, server_default=""))
    op.add_column("deployments", sa.Column("last_deployed_at", sa.DateTime(timezone=True), nullable=True))

    op.create_table(
        "deployment_snapshots",
        sa.Column("id", sa.String(36), primary_key=True),
        sa.Column("deployment_id", sa.String(36), nullable=False),
        sa.Column("captured_for_revision", sa.Integer(), nullable=False),
        sa.Column("existed", sa.Boolean(), nullable=False),
        sa.Column("api_managed", sa.Boolean(), nullable=False),
        sa.Column("compose_yaml", sa.Text(), nullable=False),
        sa.Column("compose_env_ciphertext", sa.Text(), nullable=False),
        sa.Column("reason", sa.String(64), nullable=False),
        sa.Column("restored_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["deployment_id"], ["deployments.id"], ondelete="CASCADE"),
    )
    op.create_index("ix_deployment_snapshots_deployment_id", "deployment_snapshots", ["deployment_id"], unique=False)
    op.create_index("ix_deployment_snapshots_created_at", "deployment_snapshots", ["created_at"], unique=False)


def downgrade() -> None:
    op.drop_index("ix_deployment_snapshots_created_at", table_name="deployment_snapshots")
    op.drop_index("ix_deployment_snapshots_deployment_id", table_name="deployment_snapshots")
    op.drop_table("deployment_snapshots")
    op.drop_column("deployments", "last_deployed_at")
    op.drop_column("deployments", "last_error")
    op.drop_column("deployments", "active_revision")

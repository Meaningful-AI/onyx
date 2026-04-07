"""add modal_name to llm_provider

Revision ID: c02529d3d8a4
Revises: 503883791c39
Create Date: 2026-04-06 19:39:28.924809

"""

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "c02529d3d8a4"
down_revision = "503883791c39"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "llm_provider",
        sa.Column("modal_name", sa.String(), nullable=True),
    )


def downgrade() -> None:
    op.drop_column("llm_provider", "modal_name")

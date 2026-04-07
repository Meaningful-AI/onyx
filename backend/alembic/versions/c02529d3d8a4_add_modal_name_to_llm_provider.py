"""add is_custom_provider to llm_provider

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

# These providers natively store settings in custom_config, so a non-null
# custom_config does NOT mean the provider was created via CustomModal.
NATIVE_CUSTOM_CONFIG_PROVIDERS = (
    "bedrock",
    "vertex_ai",
    "ollama_chat",
    "lm_studio",
    "litellm_proxy",
    "bifrost",
)


def upgrade() -> None:
    op.add_column(
        "llm_provider",
        sa.Column(
            "is_custom_provider",
            sa.Boolean(),
            nullable=False,
            server_default="false",
        ),
    )

    # Backfill: for providers that don't natively use custom_config,
    # a non-null custom_config means they were created via CustomModal.
    op.execute(
        sa.text(
            """
            UPDATE llm_provider
            SET is_custom_provider = true
            WHERE custom_config IS NOT NULL
              AND provider NOT IN :native_providers
            """
        ).bindparams(sa.bindparam("native_providers", expanding=True)),
        {"native_providers": list(NATIVE_CUSTOM_CONFIG_PROVIDERS)},
    )


def downgrade() -> None:
    op.drop_column("llm_provider", "is_custom_provider")

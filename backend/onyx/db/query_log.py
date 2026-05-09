from collections.abc import Mapping
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import datetime
from typing import Any
from typing import Literal
from typing import cast as typing_cast
from uuid import UUID

from sqlalchemy import cast as sa_cast
from sqlalchemy import desc
from sqlalchemy import func
from sqlalchemy import literal
from sqlalchemy import select
from sqlalchemy import String
from sqlalchemy import union_all
from sqlalchemy.orm import Session

from onyx.configs.constants import MessageType
from onyx.db.models import ChatMessage
from onyx.db.models import ChatSession
from onyx.db.models import Persona
from onyx.db.models import SearchQuery
from onyx.db.models import User

QueryLogSource = Literal["all", "chat", "search"]
QueryLogEntrySource = Literal["chat", "search"]


@dataclass(frozen=True)
class QueryLogRow:
    id: str
    source: QueryLogEntrySource
    user_email: str | None
    query: str
    time_created: datetime
    chat_session_id: UUID | None
    chat_session_name: str | None
    assistant_name: str | None


def _chat_query_select(
    start_time: datetime | None,
    end_time: datetime | None,
    user_email: str | None,
    query_text: str | None,
):
    stmt = (
        select(
            func.concat(literal("chat:"), sa_cast(ChatMessage.id, String)).label("id"),
            literal("chat").label("source"),
            User.email.label("user_email"),
            ChatMessage.message.label("query"),
            ChatMessage.time_sent.label("time_created"),
            sa_cast(ChatSession.id, String).label("chat_session_id"),
            ChatSession.description.label("chat_session_name"),
            Persona.name.label("assistant_name"),
        )
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .outerjoin(User, User.id == ChatSession.user_id)
        .outerjoin(Persona, Persona.id == ChatSession.persona_id)
        .where(ChatMessage.message_type == MessageType.USER)
        .where(ChatMessage.message != "")
    )

    if start_time:
        stmt = stmt.where(ChatMessage.time_sent >= start_time)
    if end_time:
        stmt = stmt.where(ChatMessage.time_sent <= end_time)
    if user_email:
        stmt = stmt.where(User.email.ilike(f"%{user_email}%"))
    if query_text:
        stmt = stmt.where(ChatMessage.message.ilike(f"%{query_text}%"))

    return stmt


def _search_query_select(
    start_time: datetime | None,
    end_time: datetime | None,
    user_email: str | None,
    query_text: str | None,
):
    stmt = (
        select(
            func.concat(literal("search:"), sa_cast(SearchQuery.id, String)).label("id"),
            literal("search").label("source"),
            User.email.label("user_email"),
            SearchQuery.query.label("query"),
            SearchQuery.created_at.label("time_created"),
            sa_cast(literal(None), String).label("chat_session_id"),
            sa_cast(literal(None), String).label("chat_session_name"),
            sa_cast(literal(None), String).label("assistant_name"),
        )
        .select_from(SearchQuery)
        .outerjoin(User, User.id == SearchQuery.user_id)
    )

    if start_time:
        stmt = stmt.where(SearchQuery.created_at >= start_time)
    if end_time:
        stmt = stmt.where(SearchQuery.created_at <= end_time)
    if user_email:
        stmt = stmt.where(User.email.ilike(f"%{user_email}%"))
    if query_text:
        stmt = stmt.where(SearchQuery.query.ilike(f"%{query_text}%"))

    return stmt


def _query_log_subquery(
    start_time: datetime | None,
    end_time: datetime | None,
    user_email: str | None,
    query_text: str | None,
    source: QueryLogSource,
):
    selects = []
    if source in ("all", "chat"):
        selects.append(
            _chat_query_select(
                start_time=start_time,
                end_time=end_time,
                user_email=user_email,
                query_text=query_text,
            )
        )
    if source in ("all", "search"):
        selects.append(
            _search_query_select(
                start_time=start_time,
                end_time=end_time,
                user_email=user_email,
                query_text=query_text,
            )
        )

    if len(selects) == 1:
        return selects[0].subquery()
    return union_all(*selects).subquery()


def _query_log_rows_from_mappings(
    rows: Sequence[Mapping[str, Any]],
) -> list[QueryLogRow]:
    return [
        QueryLogRow(
            id=row["id"],
            source=typing_cast(QueryLogEntrySource, row["source"]),
            user_email=row["user_email"],
            query=row["query"],
            time_created=row["time_created"],
            chat_session_id=(
                UUID(row["chat_session_id"]) if row["chat_session_id"] else None
            ),
            chat_session_name=row["chat_session_name"],
            assistant_name=row["assistant_name"],
        )
        for row in rows
    ]


def get_query_log_page(
    db_session: Session,
    page_num: int,
    page_size: int,
    source: QueryLogSource,
    start_time: datetime | None = None,
    end_time: datetime | None = None,
    user_email: str | None = None,
    query_text: str | None = None,
) -> tuple[list[QueryLogRow], int]:
    query_log = _query_log_subquery(
        start_time=start_time,
        end_time=end_time,
        user_email=user_email,
        query_text=query_text,
        source=source,
    )

    total_items = db_session.scalar(select(func.count()).select_from(query_log)) or 0

    stmt = (
        select(query_log)
        .order_by(desc(query_log.c.time_created), desc(query_log.c.id))
        .limit(page_size)
        .offset(page_num * page_size)
    )
    rows = db_session.execute(stmt).mappings().all()

    return _query_log_rows_from_mappings(rows), total_items


def get_query_log_rows_for_export(
    db_session: Session,
    source: QueryLogSource,
    start_time: datetime | None = None,
    end_time: datetime | None = None,
    user_email: str | None = None,
    query_text: str | None = None,
    limit: int = 10000,
) -> list[QueryLogRow]:
    query_log = _query_log_subquery(
        start_time=start_time,
        end_time=end_time,
        user_email=user_email,
        query_text=query_text,
        source=source,
    )

    stmt = (
        select(query_log)
        .order_by(desc(query_log.c.time_created), desc(query_log.c.id))
        .limit(limit)
    )
    rows = db_session.execute(stmt).mappings().all()

    return _query_log_rows_from_mappings(rows)

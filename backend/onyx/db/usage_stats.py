import datetime
from collections import defaultdict
from dataclasses import dataclass

from sqlalchemy import case
from sqlalchemy import cast
from sqlalchemy import Date
from sqlalchemy import func
from sqlalchemy import select
from sqlalchemy import union_all
from sqlalchemy.orm import Session

from onyx.configs.constants import MessageType
from onyx.db.models import ChatMessage
from onyx.db.models import ChatMessageFeedback
from onyx.db.models import ChatSession
from onyx.db.models import Persona
from onyx.db.models import SearchQuery
from onyx.db.models import User


@dataclass(frozen=True)
class DailyUsageStat:
    date: datetime.date
    chat_queries: int
    search_queries: int
    total_queries: int
    active_users: int


@dataclass(frozen=True)
class TopUserUsageStat:
    user_email: str
    chat_queries: int
    search_queries: int
    total_queries: int


@dataclass(frozen=True)
class TopAssistantUsageStat:
    assistant_id: int | None
    assistant_name: str
    messages: int
    unique_users: int


@dataclass(frozen=True)
class UsageStats:
    chat_queries: int
    search_queries: int
    total_queries: int
    active_users: int
    chat_sessions: int
    positive_feedback: int
    negative_feedback: int
    daily: list[DailyUsageStat]
    top_users: list[TopUserUsageStat]
    top_assistants: list[TopAssistantUsageStat]


def _chat_query_filters(start: datetime.datetime, end: datetime.datetime):
    return (
        ChatMessage.time_sent >= start,
        ChatMessage.time_sent <= end,
        ChatMessage.message_type == MessageType.USER,
        ChatMessage.message != "",
    )


def _assistant_message_filters(start: datetime.datetime, end: datetime.datetime):
    return (
        ChatMessage.time_sent >= start,
        ChatMessage.time_sent <= end,
        ChatMessage.message_type == MessageType.ASSISTANT,
    )


def _get_total_chat_queries(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> int:
    stmt = select(func.count(ChatMessage.id)).where(*_chat_query_filters(start, end))
    return db_session.scalar(stmt) or 0


def _get_total_search_queries(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> int:
    stmt = select(func.count(SearchQuery.id)).where(
        SearchQuery.created_at >= start,
        SearchQuery.created_at <= end,
    )
    return db_session.scalar(stmt) or 0


def _get_active_users(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> int:
    chat_users = (
        select(ChatSession.user_id.label("user_id"))
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .where(*_chat_query_filters(start, end))
        .where(ChatSession.user_id.is_not(None))
    )
    search_users = (
        select(SearchQuery.user_id.label("user_id"))
        .where(SearchQuery.created_at >= start)
        .where(SearchQuery.created_at <= end)
        .where(SearchQuery.user_id.is_not(None))
    )
    users = union_all(chat_users, search_users).subquery()
    stmt = select(func.count(func.distinct(users.c.user_id)))

    return db_session.scalar(stmt) or 0


def _get_chat_sessions(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> int:
    stmt = select(func.count(ChatSession.id)).where(
        ChatSession.time_created >= start,
        ChatSession.time_created <= end,
    )
    return db_session.scalar(stmt) or 0


def _get_feedback_counts(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> tuple[int, int]:
    stmt = (
        select(
            func.sum(case((ChatMessageFeedback.is_positive.is_(True), 1), else_=0)),
            func.sum(case((ChatMessageFeedback.is_positive.is_(False), 1), else_=0)),
        )
        .select_from(ChatMessageFeedback)
        .join(ChatMessage, ChatMessage.id == ChatMessageFeedback.chat_message_id)
        .where(*_assistant_message_filters(start, end))
    )
    positive, negative = db_session.execute(stmt).one()

    return positive or 0, negative or 0


def _get_daily_usage(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
) -> list[DailyUsageStat]:
    day = "day"

    daily_chat_rows = db_session.execute(
        select(
            cast(ChatMessage.time_sent, Date).label(day),
            func.count(ChatMessage.id),
        )
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .where(*_chat_query_filters(start, end))
        .group_by(cast(ChatMessage.time_sent, Date))
        .order_by(cast(ChatMessage.time_sent, Date))
    ).all()

    daily_search_rows = db_session.execute(
        select(
            cast(SearchQuery.created_at, Date).label(day),
            func.count(SearchQuery.id),
        )
        .where(SearchQuery.created_at >= start)
        .where(SearchQuery.created_at <= end)
        .group_by(cast(SearchQuery.created_at, Date))
        .order_by(cast(SearchQuery.created_at, Date))
    ).all()

    daily_chat_users = (
        select(
            cast(ChatMessage.time_sent, Date).label(day),
            ChatSession.user_id.label("user_id"),
        )
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .where(*_chat_query_filters(start, end))
        .where(ChatSession.user_id.is_not(None))
    )
    daily_search_users = (
        select(
            cast(SearchQuery.created_at, Date).label(day),
            SearchQuery.user_id.label("user_id"),
        )
        .where(SearchQuery.created_at >= start)
        .where(SearchQuery.created_at <= end)
        .where(SearchQuery.user_id.is_not(None))
    )
    daily_users = union_all(daily_chat_users, daily_search_users).subquery()
    daily_active_user_rows = db_session.execute(
        select(
            daily_users.c.day,
            func.count(func.distinct(daily_users.c.user_id)),
        )
        .group_by(daily_users.c.day)
        .order_by(daily_users.c.day)
    ).all()

    by_date: dict[datetime.date, dict[str, int]] = defaultdict(
        lambda: {"chat_queries": 0, "search_queries": 0, "active_users": 0}
    )
    for row_day, count in daily_chat_rows:
        by_date[row_day]["chat_queries"] = count or 0
    for row_day, count in daily_search_rows:
        by_date[row_day]["search_queries"] = count or 0
    for row_day, count in daily_active_user_rows:
        by_date[row_day]["active_users"] = count or 0

    return [
        DailyUsageStat(
            date=day,
            chat_queries=values["chat_queries"],
            search_queries=values["search_queries"],
            total_queries=values["chat_queries"] + values["search_queries"],
            active_users=values["active_users"],
        )
        for day, values in sorted(by_date.items())
    ]


def _get_top_users(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
    limit: int,
) -> list[TopUserUsageStat]:
    by_email: dict[str, dict[str, int]] = defaultdict(lambda: {"chat_queries": 0, "search_queries": 0})

    chat_rows = db_session.execute(
        select(User.email, func.count(ChatMessage.id))
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .join(User, User.id == ChatSession.user_id)
        .where(*_chat_query_filters(start, end))
        .group_by(User.email)
    ).all()
    search_rows = db_session.execute(
        select(User.email, func.count(SearchQuery.id))
        .select_from(SearchQuery)
        .join(User, User.id == SearchQuery.user_id)
        .where(SearchQuery.created_at >= start)
        .where(SearchQuery.created_at <= end)
        .group_by(User.email)
    ).all()

    for email, count in chat_rows:
        if email:
            by_email[email]["chat_queries"] = count or 0
    for email, count in search_rows:
        if email:
            by_email[email]["search_queries"] = count or 0

    return sorted(
        [
            TopUserUsageStat(
                user_email=email,
                chat_queries=values["chat_queries"],
                search_queries=values["search_queries"],
                total_queries=values["chat_queries"] + values["search_queries"],
            )
            for email, values in by_email.items()
        ],
        key=lambda item: item.total_queries,
        reverse=True,
    )[:limit]


def _get_top_assistants(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
    limit: int,
) -> list[TopAssistantUsageStat]:
    rows = db_session.execute(
        select(
            Persona.id,
            Persona.name,
            func.count(ChatMessage.id),
            func.count(func.distinct(ChatSession.user_id)),
        )
        .select_from(ChatMessage)
        .join(ChatSession, ChatSession.id == ChatMessage.chat_session_id)
        .outerjoin(Persona, Persona.id == ChatSession.persona_id)
        .where(*_assistant_message_filters(start, end))
        .group_by(Persona.id, Persona.name)
        .order_by(func.count(ChatMessage.id).desc())
        .limit(limit)
    ).all()

    return [
        TopAssistantUsageStat(
            assistant_id=assistant_id,
            assistant_name=assistant_name or "No assistant",
            messages=messages or 0,
            unique_users=unique_users or 0,
        )
        for assistant_id, assistant_name, messages, unique_users in rows
    ]


def get_admin_usage_stats(
    db_session: Session,
    start: datetime.datetime,
    end: datetime.datetime,
    limit: int = 10,
) -> UsageStats:
    chat_queries = _get_total_chat_queries(db_session=db_session, start=start, end=end)
    search_queries = _get_total_search_queries(
        db_session=db_session,
        start=start,
        end=end,
    )
    positive_feedback, negative_feedback = _get_feedback_counts(
        db_session=db_session,
        start=start,
        end=end,
    )

    return UsageStats(
        chat_queries=chat_queries,
        search_queries=search_queries,
        total_queries=chat_queries + search_queries,
        active_users=_get_active_users(db_session=db_session, start=start, end=end),
        chat_sessions=_get_chat_sessions(db_session=db_session, start=start, end=end),
        positive_feedback=positive_feedback,
        negative_feedback=negative_feedback,
        daily=_get_daily_usage(db_session=db_session, start=start, end=end),
        top_users=_get_top_users(
            db_session=db_session,
            start=start,
            end=end,
            limit=limit,
        ),
        top_assistants=_get_top_assistants(
            db_session=db_session,
            start=start,
            end=end,
            limit=limit,
        ),
    )

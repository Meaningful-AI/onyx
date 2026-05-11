import datetime

from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from pydantic import BaseModel
from sqlalchemy.orm import Session

from onyx.auth.permissions import require_permission
from onyx.auth.users import get_display_email
from onyx.configs.constants import PUBLIC_API_TAGS
from onyx.db.engine.sql_engine import get_session
from onyx.db.enums import Permission
from onyx.db.models import User
from onyx.db.usage_stats import DailyUsageStat
from onyx.db.usage_stats import get_admin_usage_stats
from onyx.db.usage_stats import TopAssistantUsageStat
from onyx.db.usage_stats import TopUserUsageStat
from onyx.error_handling.error_codes import OnyxErrorCode
from onyx.error_handling.exceptions import OnyxError

router = APIRouter()

_DEFAULT_LOOKBACK_DAYS = 30


class DailyUsageStatSnapshot(BaseModel):
    date: datetime.date
    chat_queries: int
    search_queries: int
    total_queries: int
    active_users: int


class TopUserUsageStatSnapshot(BaseModel):
    user_email: str
    chat_queries: int
    search_queries: int
    total_queries: int


class TopAssistantUsageStatSnapshot(BaseModel):
    assistant_id: int | None
    assistant_name: str
    messages: int
    unique_users: int


class AdminUsageStatsSnapshot(BaseModel):
    start: datetime.datetime
    end: datetime.datetime
    chat_queries: int
    search_queries: int
    total_queries: int
    active_users: int
    chat_sessions: int
    positive_feedback: int
    negative_feedback: int
    daily: list[DailyUsageStatSnapshot]
    top_users: list[TopUserUsageStatSnapshot]
    top_assistants: list[TopAssistantUsageStatSnapshot]


def _validate_time_range(
    start: datetime.datetime | None,
    end: datetime.datetime | None,
) -> tuple[datetime.datetime, datetime.datetime]:
    normalized_end = end or datetime.datetime.now(datetime.UTC)
    normalized_start = start or (normalized_end - datetime.timedelta(days=_DEFAULT_LOOKBACK_DAYS))

    if normalized_start >= normalized_end:
        raise OnyxError(
            OnyxErrorCode.INVALID_INPUT,
            "start must be before end",
        )

    return normalized_start, normalized_end


def _daily_snapshot(row: DailyUsageStat) -> DailyUsageStatSnapshot:
    return DailyUsageStatSnapshot(
        date=row.date,
        chat_queries=row.chat_queries,
        search_queries=row.search_queries,
        total_queries=row.total_queries,
        active_users=row.active_users,
    )


def _top_user_snapshot(row: TopUserUsageStat) -> TopUserUsageStatSnapshot:
    return TopUserUsageStatSnapshot(
        user_email=get_display_email(row.user_email),
        chat_queries=row.chat_queries,
        search_queries=row.search_queries,
        total_queries=row.total_queries,
    )


def _top_assistant_snapshot(
    row: TopAssistantUsageStat,
) -> TopAssistantUsageStatSnapshot:
    return TopAssistantUsageStatSnapshot(
        assistant_id=row.assistant_id,
        assistant_name=row.assistant_name,
        messages=row.messages,
        unique_users=row.unique_users,
    )


@router.get("/admin/usage-stats", tags=PUBLIC_API_TAGS)
def get_admin_usage_stats_api(
    start: datetime.datetime | None = None,
    end: datetime.datetime | None = None,
    limit: int = Query(10, ge=1, le=50),
    _: User = Depends(require_permission(Permission.FULL_ADMIN_PANEL_ACCESS)),
    db_session: Session = Depends(get_session),
) -> AdminUsageStatsSnapshot:
    normalized_start, normalized_end = _validate_time_range(start=start, end=end)
    stats = get_admin_usage_stats(
        db_session=db_session,
        start=normalized_start,
        end=normalized_end,
        limit=limit,
    )

    return AdminUsageStatsSnapshot(
        start=normalized_start,
        end=normalized_end,
        chat_queries=stats.chat_queries,
        search_queries=stats.search_queries,
        total_queries=stats.total_queries,
        active_users=stats.active_users,
        chat_sessions=stats.chat_sessions,
        positive_feedback=stats.positive_feedback,
        negative_feedback=stats.negative_feedback,
        daily=[_daily_snapshot(row) for row in stats.daily],
        top_users=[_top_user_snapshot(row) for row in stats.top_users],
        top_assistants=[_top_assistant_snapshot(row) for row in stats.top_assistants],
    )

import csv
import io
from datetime import datetime
from typing import Literal
from uuid import UUID

from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from sqlalchemy.orm import Session

from onyx.auth.permissions import require_permission
from onyx.auth.users import get_display_email
from onyx.configs.constants import PUBLIC_API_TAGS
from onyx.db.engine.sql_engine import get_session
from onyx.db.enums import Permission
from onyx.db.models import User
from onyx.db.query_log import get_query_log_page
from onyx.db.query_log import get_query_log_rows_for_export
from onyx.db.query_log import get_query_log_user_emails
from onyx.db.query_log import QueryLogRow
from onyx.db.query_log import QueryLogSource
from onyx.error_handling.error_codes import OnyxErrorCode
from onyx.error_handling.exceptions import OnyxError
from onyx.server.documents.models import PaginatedReturn

router = APIRouter()


class QueryLogEntry(BaseModel):
    id: str
    source: Literal["chat", "search"]
    user_email: str
    query: str
    time_created: datetime
    chat_session_id: UUID | None = None
    chat_session_name: str | None = None
    assistant_name: str | None = None


class QueryLogUser(BaseModel):
    email: str
    display_email: str


def _validate_date_range(start_time: datetime | None, end_time: datetime | None) -> None:
    if start_time and end_time and start_time >= end_time:
        raise OnyxError(
            OnyxErrorCode.INVALID_INPUT,
            "start_time must be before end_time",
        )


def _query_log_entry_from_row(row: QueryLogRow) -> QueryLogEntry:
    return QueryLogEntry(
        id=row.id,
        source=row.source,
        user_email=get_display_email(row.user_email),
        query=row.query,
        time_created=row.time_created,
        chat_session_id=row.chat_session_id,
        chat_session_name=row.chat_session_name,
        assistant_name=row.assistant_name,
    )


def _query_log_entries_from_rows(rows: list[QueryLogRow]) -> list[QueryLogEntry]:
    return [_query_log_entry_from_row(row) for row in rows]


@router.get("/admin/query-log/users", tags=PUBLIC_API_TAGS)
def get_admin_query_log_users(
    _: User = Depends(require_permission(Permission.FULL_ADMIN_PANEL_ACCESS)),
    db_session: Session = Depends(get_session),
) -> list[QueryLogUser]:
    return [
        QueryLogUser(email=email, display_email=get_display_email(email))
        for email in get_query_log_user_emails(db_session=db_session)
    ]


@router.get("/admin/query-log", tags=PUBLIC_API_TAGS)
def get_admin_query_log(
    page_num: int = Query(0, ge=0),
    page_size: int = Query(50, ge=1, le=200),
    source: QueryLogSource = Query("all"),
    start_time: datetime | None = None,
    end_time: datetime | None = None,
    user_email: str | None = None,
    q: str | None = None,
    _: User = Depends(require_permission(Permission.FULL_ADMIN_PANEL_ACCESS)),
    db_session: Session = Depends(get_session),
) -> PaginatedReturn[QueryLogEntry]:
    _validate_date_range(start_time=start_time, end_time=end_time)

    rows, total_items = get_query_log_page(
        db_session=db_session,
        page_num=page_num,
        page_size=page_size,
        source=source,
        start_time=start_time,
        end_time=end_time,
        user_email=user_email,
        query_text=q,
    )

    return PaginatedReturn(
        items=_query_log_entries_from_rows(rows),
        total_items=total_items,
    )


@router.get("/admin/query-log/download", tags=PUBLIC_API_TAGS)
def download_admin_query_log(
    source: QueryLogSource = Query("all"),
    start_time: datetime | None = None,
    end_time: datetime | None = None,
    user_email: str | None = None,
    q: str | None = None,
    limit: int = Query(10000, ge=1, le=100000),
    _: User = Depends(require_permission(Permission.FULL_ADMIN_PANEL_ACCESS)),
    db_session: Session = Depends(get_session),
) -> StreamingResponse:
    _validate_date_range(start_time=start_time, end_time=end_time)

    rows = get_query_log_rows_for_export(
        db_session=db_session,
        source=source,
        start_time=start_time,
        end_time=end_time,
        user_email=user_email,
        query_text=q,
        limit=limit,
    )

    output = io.StringIO()
    writer = csv.writer(output)
    writer.writerow(
        [
            "Timestamp",
            "Source",
            "User",
            "Query",
            "Chat Session ID",
            "Chat Session Name",
            "Assistant",
        ]
    )
    for entry in _query_log_entries_from_rows(rows):
        writer.writerow(
            [
                entry.time_created.isoformat(),
                entry.source,
                entry.user_email,
                entry.query,
                str(entry.chat_session_id) if entry.chat_session_id else "",
                entry.chat_session_name or "",
                entry.assistant_name or "",
            ]
        )

    csv_content = output.getvalue()
    output.close()

    return StreamingResponse(
        io.BytesIO(csv_content.encode("utf-8")),
        media_type="text/csv",
        headers={
            "Content-Disposition": 'attachment; filename="query-log.csv"',
        },
    )

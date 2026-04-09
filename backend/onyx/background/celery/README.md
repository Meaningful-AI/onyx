# Celery Development Notes

This document is the local reference for Celery worker structure and task-writing rules in Onyx.

## Worker Types

Onyx uses multiple specialized workers:

1. `primary`: coordinates core background tasks and system-wide operations.
2. `docfetching`: fetches documents from connectors and schedules downstream work.
3. `docprocessing`: runs the indexing pipeline for fetched documents.
4. `light`: handles lightweight and fast operations.
5. `heavy`: handles more resource-intensive operations.
6. `kg_processing`: runs knowledge-graph processing and clustering.
7. `monitoring`: collects health and system metrics.
8. `user_file_processing`: processes user-uploaded files.
9. `beat`: schedules periodic work.

For actual implementation details, inspect:

- `backend/onyx/background/celery/apps/`
- `backend/onyx/background/celery/configs/`
- `backend/onyx/background/celery/tasks/`

## Task Rules

- Always use `@shared_task` rather than `@celery_app`.
- Put tasks under `background/celery/tasks/` or `ee/background/celery/tasks/`.
- Never enqueue a task without `expires=`. This is a hard requirement because stale queued work can
  accumulate without bound.
- Do not rely on Celery time-limit enforcement. These workers run in thread pools, so timeout logic
  must be implemented inside the task itself.

## Testing Note

If you change Celery worker code and want to validate it against a running local worker, the worker
usually needs to be restarted manually. There is no general auto-restart on code change.

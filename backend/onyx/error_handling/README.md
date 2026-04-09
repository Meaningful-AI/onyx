# Error Handling

This directory is the local source of truth for backend API error handling.

## Primary Rule

Raise `OnyxError` from `onyx.error_handling.exceptions` instead of `HTTPException`.

The global FastAPI exception handler converts `OnyxError` into the standard JSON shape:

```json
{"error_code": "...", "detail": "..."}
```

This keeps API behavior consistent and avoids repetitive route-level boilerplate.

## Examples

```python
from onyx.error_handling.error_codes import OnyxErrorCode
from onyx.error_handling.exceptions import OnyxError

# Good
raise OnyxError(OnyxErrorCode.NOT_FOUND, "Session not found")

# Good
raise OnyxError(OnyxErrorCode.UNAUTHENTICATED)

# Good: preserve a dynamic upstream status code
raise OnyxError(
    OnyxErrorCode.BAD_GATEWAY,
    detail,
    status_code_override=e.response.status_code,
)
```

Avoid:

```python
raise HTTPException(status_code=404, detail="Session not found")
```

## Notes

- Available error codes are defined in `backend/onyx/error_handling/error_codes.py`.
- If a new error category is needed, add it there first rather than inventing ad hoc strings.
- When forwarding upstream service failures with dynamic status codes, use `status_code_override`.

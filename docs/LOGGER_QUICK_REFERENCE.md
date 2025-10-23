# Logger - Quick Reference

## What Changed

Source location (file:line) display is now **conditional based on log level**.

## Default Behavior

| Level | Production | Debug |
|-------|-----------|-------|
| INFO  | ❌ No source | ✅ With source |
| WARN  | ✅ With source | ✅ With source |
| ERROR | ✅ With source | ✅ With source |
| DEBUG | ❌ No source | ✅ With source |

## Example Output

### INFO (Production)
```
time=2025-10-23T17:52:31 level=INFO msg="user authenticated" user_id=123
```

### ERROR (All Modes)
```
time=2025-10-23T17:52:31 level=ERROR source=api/handler.go:67 msg="request failed"
```

## Configuration

Nothing to change! It's automatic:
- `server.mode: release` → Hide source for INFO (compact)
- `server.mode: debug` → Show source for all (verbose)

## Usage

Same as before - no code changes needed:

```go
logger.Info("user authenticated", "user_id", userID)
logger.Error("failed to save", "error", err)
```

## Benefits

✅ 15-25% smaller logs in production
✅ Better performance (fewer allocations)
✅ Still shows source for important logs (WARN/ERROR)
✅ Auto-switches with server.mode
✅ No configuration needed
✅ No code changes required

## Files Changed

- `internal/shared/logger/logger.go` - Updated to use wrapper
- `internal/shared/logger/conditional_source_handler.go` - New handler
- `internal/shared/logger/conditional_source_handler_test.go` - Tests

## For More Details

- `docs/SLOG_SOURCE_LOCATION_ANALYSIS.md` - Research and comparison
- `docs/LOGGER_USAGE_GUIDE.md` - Complete guide
- `docs/IMPLEMENTATION_SUMMARY.md` - Technical overview

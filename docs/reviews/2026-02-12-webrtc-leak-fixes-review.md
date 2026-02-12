# Code Review: WebRTC Resource Leak Fixes (Partial)

**Reviewed:** Tasks 1-3 from docs/plans/2026-02-12-webrtc-resource-leak-fixes.md
**Date:** 2026-02-12T19:30:00Z
**Range:** 1a8aca5..a122ffd

## Summary

Partial implementation of WebRTC resource leak fixes completed. Successfully added defer cleanup to `CreateOffer()` and `AcceptOffer()`, plus tests. However, code review identified several issues that need addressing before merging.

---

## Strengths

- ✅ Correct defer cleanup pattern implemented
- ✅ Proper ICE gathering timeout handling
- ✅ Good error wrapping with `%w`
- ✅ Added `shutdown()` method for cleanup
- ✅ All tests passing, no regressions

---

## Issues

### Important Issues

#### 1. AcceptOffer ICE timeout missing explicit Close
- **File:** `app.go:216-217`
- **Issue:**
  ```go
  case <-time.After(TimeoutWebRTCICE):
      return "", fmt.Errorf("ICE gathering timeout: failed to gather candidates after %v", TimeoutWebRTCICE)
  ```
  No explicit `peerConnection.Close()` before return, unlike `CreateOffer()`.
- **Why it matters:** Inconsistent error handling. If `cleanupNeeded` is somehow false before this point, connection leaks.
- **Fix:**
  ```go
  case <-time.After(TimeoutWebRTCICE):
      peerConnection.Close()
      return "", fmt.Errorf("ICE gathering timeout: failed to gather candidates after %v", TimeoutWebRTCICE)
  ```

#### 2. AcceptOffer marshal error double-close risk
- **File:** `app.go:224`
- **Issue:**
  ```go
  if err != nil {
      peerConnection.Close()  // Explicit close
      return "", fmt.Errorf("failed to marshal answer: %w", err)
  }
  ```
  When `cleanupNeeded` is `true` (still is here), defer will also close.
- **Why it matters:** Double-close is semantically incorrect, could hide bugs.
- **Fix:**
  ```go
  if err != nil {
      cleanupNeeded = false
      peerConnection.Close()
      return "", fmt.Errorf("failed to marshal answer: %w", err)
  }
  ```

#### 3. CreateOffer marshal error double-close risk
- **File:** `app.go:142`
- **Issue:** Same as #2 above
- **Why it matters:** Same semantic issue
- **Fix:**
  ```go
  if err != nil {
      cleanupNeeded = false
      peerConnection.Close()
      return "", fmt.Errorf("failed to marshal offer: %w", err)
  }
  ```

#### 4. Test coverage gap - Missing error injection tests
- **File:** `app_test.go:89-118`
- **Issue:** Tests don't verify cleanup on actual error paths. Only test happy paths.
- **Why it matters:** Without error injection, can't verify defer cleanup works when errors occur.
- **Fix:** Add tests for:
  - `NewPeerConnection` failure
  - `CreateDataChannel` failure
  - ICE gathering timeout
  - Marshal errors

#### 5. Test coverage gap - Missing ICE timeout test
- **File:** N/A (test doesn't exist)
- **Issue:** No test verifies ICE gathering timeout properly closes connection.
- **Why it matters:** Critical error path should be tested.
- **Fix:** Add test mocking ICE gathering to timeout and verifying connection closure.

#### 6. Test name misleading
- **File:** `app_test.go:107`
- **Issue:** `TestCreateOfferHandlesCreateOfferError` suggests it tests error path, but only tests happy path.
- **Why it matters:** Misleading names make codebase harder to understand.
- **Fix:** Rename to `TestCreateOfferSetsPeerConnectionAndReturnsOffer` or update test.

### Minor Issues

#### 7. Missing integration test for connection cycles
- **File:** N/A (test doesn't exist)
- **Issue:** Task 8's `TestMultipleConnectionCycles` not implemented.
- **Why it matters:** Can't verify repeated CreateOffer/shutdown cycles don't leak resources.
- **Fix:** Add test from plan (Task 8).

#### 8. Scope creep - File I/O timeout changes
- **File:** `app.go:340-366`
- **Issue:** Added timeout handling to `ExportToFile()` and `ImportFromFile()` - not in Tasks 1-3.
- **Why it matters:** Increases scope and complexity of change.
- **Fix:** Revert or document as separate improvement.

#### 9. Scope creep - timeout.go changes
- **File:** `timeout.go`
- **Issue:** Added `DialTimeout`, `ListenTimeout`, constants - not in Tasks 1-3.
- **Why it matters:** Adds complexity beyond planned scope.
- **Fix:** Document as infrastructure improvements or separate into different PR.

---

## Recommendations

1. **Fix double-close semantic issues** in marshal error paths
2. **Add explicit Close** in AcceptOffer ICE timeout path for consistency
3. **Add error injection tests** to verify defer cleanup works on error paths
4. **Add integration test** for multiple connection cycles
5. **Rename misleading test** to accurately reflect what it tests
6. **Consider scope separation:** File I/O and network timeout improvements could be separate PR

---

## Assessment

**Ready to merge:** No - With fixes

**Reasoning:** Core defer cleanup implementation is correct, but semantic issues (double-close), missing test coverage for error paths, and inconsistent error handling need addressing. Fixes are straightforward.

---

## Next Steps

1. Fix Important issues #1-6 (code and tests)
2. Decide on Minor issues #7-9 (fix or document)
3. Re-request code review after fixes
4. Continue with remaining tasks (4-11) from the plan

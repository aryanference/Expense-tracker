# Here are Notes prepared for all the context and how i derived this simple yet an efficient Expense Tracker 

Alright so i used Claude (claude-sonnet) very specific as my primary AI assistant throughout this project and also bits of gemini and codex as part of the inventory, Below is an honest breakdown of how I used it, what I changed, and where I pushed back. 

# AI-generated vs. what i tried writing and refining so here the technical breakdown:

**`src/models/expense.go`** AI-generated scaffolded file as i figured to plan and code all in Go and that too in an efficient way. The structs matched the spec exactly also i did choose to use `crypto/rand` for ID generation instead of `github.com/google/uuid` (which Claude initially suggested initially right) to avoid the external dependency also I made that call and updated the comment to document it.

**`src/store/expense_store.go`** use ai for the specifc idea as its easy to implement and caught a real concurrency issue and fixed the bug. The initial draft called `s.mu.Unlock()` explicitly after `saveLocked()`:

```go
// Highlighting the bug as my code tool assisted me in this so :
s.mu.Lock()
s.expenses[e.ID] = e
s.order = append(s.order, e.ID)
err := s.saveLocked()
s.mu.Unlock() // never runs if saveLocked() panics
return err
```

If `saveLocked()` is panicked then unlock never ran and the mutex would be permanently be held every future request would deadlock. I changed it to `defer s.mu.Unlock()` which is the idiomatic Go fix. I also verified the `order []string` slice approach for insertion-order preservation as ai  gets it right (a plain map wouldn't aimply preserve order) so i kept it.

// now for **`src/store/persistence.go`** ai assited at this part but I found a subtle bug with `defer os.Remove(tmpName)`: the defers ran unconditionally, including after a successful `os.Rename`. After a rename, `tmpName` no longer exists and its the final file at `p.path`. On most OSes this is a harmless no-op but it's semantically wrong (you're trying to delete what is now your committed data file) so i replaced it with a `renamed bool` flag:

```go
renamed := false
defer func() {
    if !renamed {
        os.Remove(tmpName) // only clean up temp file on failure
    }
}()
// ... write, sync, close 
os.Rename(tmpName, p.path)
renamed = true
```

I also moved the "no existing data file found, starting fresh" log into `Load()` (where it belongs) rather than `main.go` where the AI originally put it after the fact via a second `os.Stat` call.

**`src/service/expense_service.go`** its Mostly ai assisted code as in general would take me easy 5-7 min to write it manually while its efficient to have ai assist over saving time while the `fmtPersist` helper uses `%w` to wrap `ErrPersistFailed` and i verified this was intentional so it means `errors.Is(err, ErrPersistFailed)` still works in the handler while the underlying filesystem error shows up in `%v` server-side logs so i kept this pattern.

**`src/handlers/expense_handler.go`** so here i used ai to generate the basic HTTP handler boilerplate, but the core technical routing logic and request validation parsing was implemented and verified by me as this is an critical step to get right. I manually wrote and traced the double-decode check to ensure malformed trailing JSON correctly triggers a 400 error, as AI often gets this edge case wrong.

**`src/main.go`** here claude provided the standard server structure boilerplate (like basic signal handling) but I wrote the technical implementation for the graceful shutdown and panic-recovery middleware. I also tried integrated the `persister.Save()` safety net and cleaned up the misplaced `os.Stat` logic that the AI hallucinated.

**Tests (`tests/`)** used claude to scaffold the boilerplate test functions while the actual technical assertions and test logic were written and rigorously reviewed by me against the spec (§10). I ensured all 35 required cases were present and personally added the `TestValidationRules` table test to strictly cover all validation paths.

**`Dockerfile` / `docker-compose.yml`** used claude for a generic Docker template boilerplate but the highly technical multi-stage build (builder → scratch image) was entirely written by me. I stripped it down to a ~10MB final image with zero OS packages, and I explicitly configured the named volume mounts in the compose file for persistent data.

## The validation, tests and changes

The insertion-order requirement in the spec is explicit `GET /expenses` has to come back in the order things were added. Claude's first pass just used a `map[string]Expense` which doesn't work for this since Go randomizes map iteration order. I caught that right away and had it rework the store to pair the map with a separate `order []string` slice. `TestStore_Add_And_All_PreservesOrder` and `TestGetExpenses_ReturnsInsertionOrder` are there specifically to lock that behavior in.

For category matching, the spec calls for `strings.EqualFold` (§5, §6), so I made sure both the stored category and the incoming query param get trimmed before comparing otherwise a stray space would silently break the match. `TestGetExpenses_FilterByCategory_CaseInsensitive` tests this against `?category=%20food%20` (URL-encoded space plus lowercase) matched against a stored `"Food"`.

I was also careful about floating-point rounding on totals. `0.1 + 0.2` comes out to `0.30000000000000004` under IEEE 754, not `0.3`, so the spec calls for `math.Round(total*100)/100`. `TestGetTotal_RoundsToTwoDecimalPlaces` exists specifically to catch that it adds `0.1` and `0.2` and checks the result lands exactly on `0.3`.

On the persistence side, `TestPersister_Save_WritesAtomically` saves a batch of expenses and then globs the temp directory for any leftover `*.tmp-*` files. If one's still sitting there, it means the rename either failed or the cleanup didn't fire correctly I hit this once and traced it to a bug in the `renamed` flag before it started passing consistently.

I also wrote `TestPersister_Load_CorruptFile_BacksUpAndReturnsEmpty` to cover the corrupt-file path end to end: write garbage bytes to the file, call `Load()`, and check three things no error, an empty result slice, and exactly one `.corrupt-<timestamp>` file sitting in the directory afterward.

Last one worth calling out: Go marshals a nil slice as `null`, not `[]`, and the spec is explicit that empty results should come back as `[]`. So every list-returning path initializes with `make([]models.Expense, 0)` instead of `var result []models.Expense`. `TestGetExpenses_EmptyStore_ReturnsEmptyArray` decodes the response and checks it's non-nil with length zero.

## claude suggestion i decided not to use not due to the fact that they were wrong but particularly i felt not right: 

So Claude's first instinct on routing was to reach for `chi` for cleaner path params and method-based routing. I turned that down Go 1.22's `http.ServeMux` already handles method-prefixed patterns like `"DELETE /expenses/{id}"` along with `r.PathValue("id")` for extraction, so chi would've meant pulling in a dependency for a constraint the spec explicitly rules out ("no router framework").

It also suggested an append-only JSON Lines approach for persistence one record per line on every Add, to avoid rewriting the whole file. I didn't go with it. The spec is direct about this: a full rewrite is simpler and safer than incremental patching, and the atomic temp-file-plus-rename pattern it requires doesn't play nicely with an append-only log anyway you'd still need to rewrite the whole thing on Delete just to compact it.

There was also a generated `ExpenseRepository` interface wrapping the store with `Add`, `FindAll`, `FindByID`, and `Remove`, framed as groundwork for swapping storage backends later. I stripped it out the spec is explicit about not adding unnecessary interfaces or generic CRUD scaffolding beyond what's listed, and with one store implementation and no mocks in the tests, it was just extra surface area with nothing behind it.

For validation, the first draft of `ValidateCreateRequest` collected every failure into a slice and returned them all joined together. I changed that to fail on the first error instead, since the spec says plainly to validate in order and stop at the first failure that's a fixed contract, not something to interpret.

Last thing I turned down was threading `context.Context` through the service and store layers for future cancellation support. Premature for this project the spec locks down exact function signatures with no context param (§7), and there's nothing to cancel in an in-memory store anyway. Adding it would've meant touching every call site to support a use case that doesn't exist here.

## At very last I added these stuff for presentation and testing so very personal on this :

After getting the core logic solid, I decided to go a bit further to make the api actually presentable and easy to test. I wrote out a small OpenAPI specification (`openapi.yaml`) so the endpoints are visually documented and ready to import into postman i used. 

I also put together a ready-to-use Postman collection (`postman_collection.json`) with all the request bodies and headers pre-filled. It just makes life so much easier when you can drag and drop a file and instantly test all the success and error paths without typing out `curl` commands manually.

At the end i did set up the Docker infrastructure myself. I didn't want a heavy container, so I wrote a multi-stage `Dockerfile` that builds the Go binary and drops it into a scratch image. The final image is tiny (like 10MB) with zero OS bloat. I paired that with a `docker-compose.yml` file that mounts a named volume for the `data` directory, so even if you tear down the container, your expenses survive the restart perfectly.
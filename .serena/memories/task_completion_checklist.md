# Task Completion Checklist

When completing any coding task, follow this checklist to ensure quality and consistency.

## Definition of Done

Before considering a task complete, verify:

- [ ] **Tests written and passing**
  - Run `go test ./...` for backend changes
  - All existing tests still pass
  - New functionality has test coverage
  - Tests follow existing patterns

- [ ] **Code follows project conventions**
  - Reviewed code style guidelines in CLAUDE.md
  - Matches patterns from similar existing code
  - Proper error handling implemented
  - Clear variable and function names

- [ ] **No linter/formatter warnings**
  - Backend: `gofmt -w .` applied
  - Backend: `go vet ./...` passes
  - Frontend: `npm run lint` passes (if frontend changes)

- [ ] **Code compiles successfully**
  - Backend: `go build` succeeds
  - Frontend: `npm run build` succeeds (if frontend changes)
  - No compilation errors or warnings

- [ ] **Commit messages are clear**
  - Message explains "why" not just "what"
  - References issue numbers if applicable
  - Follows project commit conventions

- [ ] **Implementation matches plan**
  - If IMPLEMENTATION_PLAN.md exists, status updated
  - Deliverables from plan are met
  - Success criteria satisfied

- [ ] **No TODOs without issue numbers**
  - All TODO comments have tracking references
  - Or TODOs are removed/resolved

## Commands to Run

### After Backend Changes

1. **Format code**:
   ```bash
   cd backend
   gofmt -w .
   ```

2. **Run static analysis**:
   ```bash
   cd backend
   go vet ./...
   ```

3. **Run tests**:
   ```bash
   cd backend
   go test ./...
   ```

4. **Build to verify**:
   ```bash
   cd backend
   go build cmd/server/main.go
   ```

### After Frontend Changes

1. **Run linter**:
   ```bash
   cd frontend
   npm run lint
   ```

2. **Build to verify**:
   ```bash
   cd frontend
   npm run build
   ```

### Before Committing

1. **Self-review changes**:
   ```bash
   git diff
   ```

2. **Stage appropriate files**:
   ```bash
   git add <files>
   ```

3. **Commit with clear message**:
   ```bash
   git commit -m "Clear description of change and why"
   ```

## Important Reminders

**NEVER**:
- Use `--no-verify` to bypass commit hooks
- Disable tests instead of fixing them
- Commit code that doesn't compile
- Make assumptions - verify with existing code
- Introduce new tools without strong justification

**ALWAYS**:
- Commit working code incrementally
- Update plan documentation as you go
- Learn from existing implementations
- Stop after 3 failed attempts and reassess
- Test behavior, not implementation
- Keep tests deterministic

## Decision Framework

When multiple valid approaches exist, choose based on:

1. **Testability** - Can I easily test this?
2. **Readability** - Will someone understand this in 6 months?
3. **Consistency** - Does this match project patterns?
4. **Simplicity** - Is this the simplest solution that works?
5. **Reversibility** - How hard to change later?

## Test Guidelines

- Test behavior, not implementation
- One assertion per test when possible
- Clear test names describing scenario
- Use existing test utilities/helpers
- Tests should be deterministic
- Use `t.TempDir()` for test isolation (Go)

## When Stuck

After 3 attempts on the same issue:

1. Document what failed (errors, approaches tried)
2. Research alternatives (find 2-3 similar implementations)
3. Question fundamentals (is this the right abstraction?)
4. Try different angle (different library/pattern/simpler approach)

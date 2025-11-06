# Code Style and Conventions

## Go Backend Style

### General Principles (from CLAUDE.md)
- **Incremental progress over big bangs** - Small changes that compile and pass tests
- **Learning from existing code** - Study and plan before implementing
- **Pragmatic over dogmatic** - Adapt to project reality
- **Clear intent over clever code** - Be boring and obvious
- **Single responsibility** per function
- **Avoid premature abstractions**
- **No clever tricks** - choose the boring solution

### Go Conventions

#### Naming
- Package names: lowercase, single word (e.g., `config`, `scraper`)
- Exported identifiers: CamelCase (e.g., `Load`, `ServiceConfig`)
- Unexported identifiers: camelCase (e.g., `parseConfig`)
- Interfaces: typically named with "-er" suffix when appropriate

#### Error Handling
- Fail fast with descriptive messages
- Include context for debugging
- Handle errors at appropriate level
- Never silently swallow exceptions
- Use explicit error checks: `if err != nil`

#### Testing
- Test files: `*_test.go`
- Test functions: `func Test<Name>(t *testing.T)`
- Use `t.TempDir()` for test isolation
- Clear test names describing scenario
- Table-driven tests when appropriate
- One logical assertion per test when possible
- Tests should be deterministic

#### Code Organization
- Use internal packages for non-exported code
- Group related functionality in packages
- Keep cmd/ lightweight - delegate to internal/
- Composition over inheritance
- Interfaces over singletons
- Explicit over implicit dependencies

#### Comments
- Document exported functions, types, and packages
- Comments explain "why", not "what"
- Test comments describe setup steps

## Frontend Style

### React/JavaScript Conventions
- Component files: `.jsx` extension
- Components: PascalCase (e.g., `ServiceCard.jsx`)
- Utilities: camelCase (e.g., `formatDuration.js`)
- Use functional components with hooks
- Follow ESLint rules (configured in `eslint.config.js`)

### Styling
- Tailwind CSS utility classes
- Responsive design patterns
- Consistent color scheme based on service branding

## File Formatting

### Go
- Use `gofmt` for formatting (standard Go formatting)
- Run before committing

### JavaScript/JSX
- ESLint configured in project
- Run `npm run lint` to check
- Vite dev server shows lint errors

## Configuration Files
- YAML for config files (config.yaml)
- Consistent indentation (2 spaces for YAML, tabs for Go)
- Comments for non-obvious settings

## Documentation
- README.md - User-facing setup and usage
- IMPLEMENTATION_PLAN.md - Technical decisions and stages
- CLAUDE.md - Development process guidelines
- Code comments - Complex logic explanation

## Important Principles from CLAUDE.md

### Before Implementation
1. Understand existing patterns in codebase
2. Write test first (TDD when possible)
3. Minimal code to pass tests
4. Refactor with tests passing
5. Commit with clear message

### When Stuck (After 3 Attempts)
- **CRITICAL**: Maximum 3 attempts per issue, then STOP
- Document what failed
- Research alternatives
- Question fundamentals
- Try different angle

### Quality Gates
- Tests written and passing
- Code follows project conventions
- No linter/formatter warnings
- Clear commit messages
- Implementation matches plan
- No TODOs without issue numbers

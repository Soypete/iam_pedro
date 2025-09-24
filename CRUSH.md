# CRUSH.md - Codebase Commands & Conventions

## Build/Test/Lint Commands
```bash
go build -v -o pedro ./cli/discord    # Build Discord bot
go build -v -o pedro ./cli/twitch     # Build Twitch bot
go test ./... -v -cover               # Run all tests with coverage
go test -run TestName ./package       # Run single test (e.g., go test -run Test_CleanResponse ./ai)
golangci-lint run                     # Lint code (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
```

## Code Style Guidelines

### Import Ordering
1. Standard library packages
2. Empty line
3. External dependencies (e.g., discordgo, sqlx)
4. Empty line
5. Internal packages (github.com/Soypete/twitch-llm-bot/...)

### Error Handling
- Wrap errors with context: `fmt.Errorf("description: %w", err)`
- Log errors before returning: `logger.Error("failed to X", "error", err.Error())`
- Return early on errors

### Naming Conventions
- Packages: lowercase, no underscores (e.g., `discordchat`)
- Exported types/funcs: PascalCase (e.g., `Client`, `NewClient`)
- Private funcs: camelCase (e.g., `handleMessage`)
- Test functions: `Test_functionName` or `TestFunctionName`
- Files: snake_case (e.g., `discord_messages.go`)

### Testing
- Use table-driven tests with `t.Run()` for subtests
- Test files alongside implementation (e.g., `ask.go` → `ask_test.go`)

### Project Structure
- CLI entrypoints: `/cli/{service}/`
- Business logic: `/ai/`, `/twitch/`, `/discord/`
- Database: `/database/` (uses sqlx, migrations in `/database/migrations/`)
- Shared types: `/types/`
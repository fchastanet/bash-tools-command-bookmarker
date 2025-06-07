---
applyTo: '**/*.go'
---
# Golang Instructions

- Use the `go` command to run, test, and build Go applications.
- Use `go run` to execute Go files directly.
- Use `go test` to run tests in the current package.
- Use `go build` to compile the application.
- avoid using `fmt.Println` for logging but prefer `slog.Info`, `slog.Error`, etc.
- Use `slog` for structured logging.
- Follow Go's idiomatic error handling pattern:
  - Check for errors immediately after calling a function that returns an error.
  - Use `if err != nil { return err }` to propagate errors.
- Use DRY (Don't Repeat Yourself) principles:
  - Avoid duplicating code; extract common functionality into functions or methods.
  - Use interfaces to define behavior and allow for easier testing and mocking.
- Use clean code practices:
  - Use meaningful variable and function names.
  - Keep functions small and focused on a single task.
  - Use comments to explain complex logic, but avoid obvious comments.
  - Use consistent naming conventions for variables and functions.
  - Avoid premature optimization; focus on writing correct and clear code first.
- Generate code so it could be easily mocked for testing purposes.
- Update Markdown files with relevant information when necessary.
- Generate unit tests for new features or changes.

# Example Workflows

This directory contains example GitHub Actions workflow files for demonstrating `gh-wf-to-mmd`.

## Examples

### `simple.yml`
A basic linear workflow demonstrating:
- Sequential job dependencies (`build` → `test` → `deploy`)
- Simple dependency chain

**Try it:**
```bash
gh wf-to-mmd examples/simple.yml
```

### `complex.yml`
A more complex workflow demonstrating:
- Parallel jobs (`lint`, `test-unit`, `test-integration`)
- Fan-in pattern (multiple jobs feeding into `build`)
- Fan-out pattern (`build` feeding into multiple deploy jobs)
- Multiple dependencies

**Try it:**
```bash
gh wf-to-mmd examples/complex.yml
gh wf-to-mmd --orientation LR examples/complex.yml
```

## Usage

From the project root:

```bash
# Build the tool
go build

# Try the examples
./gh-wf-to-mmd examples/simple.yml
./gh-wf-to-mmd --orientation LR examples/complex.yml
./gh-wf-to-mmd --output diagram.mmd examples/simple.yml
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Build and test
go build && ./gh-wf-to-mmd examples/simple.yml
```

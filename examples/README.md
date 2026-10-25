# Examples

Sample workflows and the diagrams `gh wf2mmd` generates for them. The `.md` files render as diagrams on GitHub.

| Workflow | `simple` | `detailed` |
|----------|----------|------------|
| [simple.yml](simple.yml): three jobs in a chain | [simple.md](simple.md) | [simple-detailed.md](simple-detailed.md) |
| [complex.yml](complex.yml): parallel jobs, a matrix, a reusable workflow, and conditional deployments to environments | [complex.md](complex.md) | [complex-detailed.md](complex-detailed.md) |

Try them from the repository root:

```bash
go build
./gh-wf2mmd examples/complex.yml
./gh-wf2mmd examples/complex.yml --level detailed --direction LR
./gh-wf2mmd examples/complex.yml -o diagram.md
```

The diagrams double as golden files for the tests (`TestExamples` in [main_test.go](../main_test.go)). After an intended change to the output, regenerate them with:

```bash
go test . -update
```

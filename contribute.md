# Contributing to log

Thanks for your interest in contributing to `log`.

## How To Contribute

1. Fork the repository and clone it:
```bash
git clone https://github.com/jeffotoni/log.git
cd log
```

2. Create a branch:
```bash
git checkout -b feat/short-description
```

3. Make your changes.

4. Run validations before opening a PR:
```bash
go test ./...
go test -race ./...
go test -cover ./...
```

5. Commit with a clear message:
```bash
git commit -m "feat: improve JSON/text logging behavior"
```

6. Push your branch and open a Pull Request:
```bash
git push origin feat/short-description
```

## Guidelines

- Keep PRs small and focused.
- Keep backward compatibility whenever possible.
- Add or update tests for behavior changes.
- Update `README.md` when public API/behavior changes.
- Include benchmark evidence for performance-sensitive changes.

## Reporting Issues

Please use the issues page:
[https://github.com/jeffotoni/log/issues](https://github.com/jeffotoni/log/issues)

When possible, include:
- Go version
- OS/architecture
- Minimal reproducible example
- Expected vs actual behavior

## Code Style

- Follow standard Go style.
- Prefer clear naming over clever code.
- Keep APIs simple and explicit.

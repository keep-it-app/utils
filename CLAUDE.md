# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ ВАЖНЫЕ ИНСТРУКЦИИ ⚠️

- **НИКОГДА** не упоминай Claude, Claude Code или Anthropic в сообщениях коммитов или генерируемом коде
- **НИКОГДА** не добавляй теги вроде "Generated with Claude Code" ни в какие материалы
- **ВСЕГДА** перед коммитом прогоняй линтеры и тесты; не коммить, если они не проходят
- **ВСЕГДА** пиши тесты на весь новый функционал
- **ВСЕГДА** коммить и пушь все изменения в конце работы; не упоминай соавторство Claude

## Commands

```bash
# Run all tests
go test ./...

# Lint
golangci-lint run ./...
```

## Repository Purpose

Shared Go utilities (`github.com/keep-it-app/utils`) used across all Keep-It microservices. Published on GitHub (public module).

```
logger/           — slog-based structured logger setup
logger/httpmiddleware/ — HTTP middleware for request logging (chi-compatible)
```

## Rules

- This is a public shared library — keep the API surface minimal and stable.
- Breaking changes to exported types or function signatures require a major version bump.
- All exported symbols must have GoDoc comments.
- All code, comments, and names in English.

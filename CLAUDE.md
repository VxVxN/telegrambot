# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Run from project root
BOT_TOKEN="<token>" go run ./cmd/bot

# Build binary
go build -o telegrambot ./cmd/bot
```

Requires Go 1.24+. The `BOT_TOKEN` environment variable is mandatory (Telegram BotFather token).

## Project Structure

```
cmd/bot/main.go                — entry point, wiring dependencies
internal/
  model/todo.go                — domain types: Todo, UserTodos
  currency/client.go           — HTTP client for CoinGecko and CBR APIs
  storage/
    todo.go                    — TodoStore: thread-safe todo CRUD with JSON persistence
    subscriber.go              — SubscriberStore: thread-safe subscriber management
  bot/
    bot.go                     — Bot struct, constructor, handler registration
    router.go                  — text message routing (bilingual: Russian/English)
    todo_handler.go            — todo command handlers
    price_handler.go           — price/currency handlers, daily notification loop
```

## Architecture

- **Dependency flow:** `cmd/bot` → `internal/bot` → `internal/{storage, currency, model}`
- **Concurrency:** each store has its own `sync.Mutex`. No shared locks between packages.
- **Persistence:** JSON files (`todos.json`, `subscribers.json`) in working directory. `TodoStore.nextID` is restored from loaded data on startup.
- **Background goroutine** in `price_handler.go` sleeps until 10:00 AM daily and sends price notifications to subscribers.
- **External APIs:** CoinGecko (crypto prices), CBR XML Daily (USD/RUB rate). Single shared `http.Client` per `currency.Client`.

## No Tests

There are no automated tests in this project.

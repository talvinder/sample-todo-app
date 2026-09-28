# todo-stack

Minimal todo app: React (Vite) → Go API → MySQL, with Redis as a read cache.

## Run everything

    docker compose up --build

- App: http://localhost:3000
- API: http://localhost:8080/api/todos

## How the cache works

`GET /api/todos` checks Redis key `todos:all` first (response header `X-Cache: HIT`).
On a miss it reads MySQL, stores the result in Redis for 60s (`X-Cache: MISS`).
Any add / toggle / delete deletes the key, so the next read comes fresh from MySQL.

## API

| Method | Path | Body |
|---|---|---|
| GET | /api/todos | – |
| POST | /api/todos | `{"title": "..."}` |
| PATCH | /api/todos/{id} | – (toggles done) |
| DELETE | /api/todos/{id} | – |

## Local dev without Docker for the app code

    docker compose up mysql redis
    cd backend && REDIS_ADDR=localhost:6380 go run .
    cd frontend && npm install && npm run dev   # http://localhost:5173

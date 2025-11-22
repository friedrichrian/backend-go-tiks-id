# Tiks-ID Backend (Go)

This repository provides a Go implementation of the Tiks-ID backend API. The server uses Gin + GORM and provides endpoints for authentication, movies, genres, theaters, schedules and ticket bookings.

This README documents each API endpoint, request/response shapes, authentication, file uploads (poster), pagination and run instructions.

---

**Quick Start**

- Copy `.env.example` to `.env` and update DB credentials (or set env vars). Example env keys used:
  - `DB_USER`, `DB_PASS`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `APP_PORT`
- Run migrations & seed automatically at startup (AutoMigrate + seed function are called in `cmd/api/main.go`).
- Install deps and run:

```bash
cd backend-go
go mod tidy
go run ./cmd/api
```

The server serves poster files (uploaded images) at `/posters/<filename>` (static file server configured in `cmd/api/main.go`).

If running inside Docker and connecting to host MySQL, make sure you add host mapping (or use networked DB).

---

**Authentication**

All protected endpoints require a Bearer access token in the `Authorization` header:

Header:

```
Authorization: Bearer <ACCESS_TOKEN>
```

Endpoints:

- `POST /auth/register` — register new user
  - Body JSON: `{ "fullname": "...,", "email":"...", "password":"..." }`
  - Response: created user (no token) or appropriate error.

- `POST /auth/login` — login and receive tokens
  - Body JSON: `{ "email": "...", "password": "..." }`
  - Response: `{ "access_token": "...", "refresh_token": "...", "user": { ... } }`

- `POST /auth/refresh` — rotate refresh token and get a new access token
  - Body JSON: `{ "refresh_token": "..." }`

- `POST /auth/logout` (protected) — revoke refresh token

- `GET /auth/me` (protected) — get currently authenticated user

---

**Movies**

Base route: `/movie`

1) GET `/movie` — list movies (paginated)
   - Query params:
     - `page` (default 1)
     - `per_page` (default 10, max 100)
   - Response:
     ```json
     {
       "message":"Movies found",
       "data": [ { /* MovieResponse */ } ],
       "meta": { "page":1, "per_page":10, "total":100, "total_pages":10 }
     }
     ```
   - `MovieResponse` fields: `id, title, description, duration, release_date, poster (URL), genre ([]string), created_at, updated_at`.

2) GET `/movie/:id` — show movie details (grouped schedules)
   - Protected (requires Authorization header)
   - Response `data` contains:
     - `title`, `description`, `genres` (array of strings), `duration`, `release_date`, `poster` (accessible URL), `available_theaters` grouped as:

```json
"available_theaters": [
  {
    "theater_id": 1,
    "theater": "A1",
    "section": 0,
    "row": 5,
    "column": 10,
    "available_dates": [
      {
        "date": "02 Jan 2006",
        "available_times": [
          { "schedule_id": 3, "time": "15:00", "filled_seats": ["A1","A2"] }
        ]
      }
    ]
  }
]
```

3) POST `/movie` — create movie
   - Protected
   - Supports both JSON and multipart/form-data (for file upload). For file upload use `Content-Type: multipart/form-data` and `poster` field.
   - JSON example:
     ```json
     {
       "title": "My Movie",
       "description": "...",
       "duration": 120,
       "release_date": "2008-07-18 00:00:00",
       "genre_ids": [1,2]
     }
     ```
   - Multipart form example (poster upload + genre list as comma-separated IDs):
     - `title=...`, `description=...`, `duration=120`, `release_date=2008-07-18 00:00:00`, `genre=1,2`, `poster=@/path/to/file.jpg`
   - Returns `201 Created` with the created movie data. If `title` is already used, returns `409 Conflict`.

4) PATCH `/movie/:id` — update movie (partial)
   - Protected
   - Supports multipart/form-data (so you can PATCH with `poster` file) and JSON partial updates.
   - Multipart: only provided fields are updated. `genre` is a comma-separated list of genre IDs. Poster file replaces the stored poster file; old file is removed.
   - JSON: you can send partial fields e.g. `{ "title": "New Title", "genre_ids": [1,2] }`.
   - Title uniqueness enforced — returns `409 Conflict` if attempted title exists on another movie.

5) DELETE `/movie/:id` — delete movie
   - Protected
   - Removes DB record and attempts to remove poster file on disk.

---

**Genres**

Base route: `/genre`

- `GET /genre` — list genres (protected)
- `POST /genre` — create (protected)
- `PATCH /genre/:id` — update (protected)
- `DELETE /genre/:id` — delete (protected)

Genres appear in movie responses as an array of strings (genre names).

---

**Theaters**

Base route: `/theater`

- `GET /theater` — list theaters (protected)
- `POST /theater` — create (admin only)
- `PATCH /theater/:id` — edit (admin only)
- `DELETE /theater/:id` — delete (admin only)

Theater fields: `id, name, section, row, col` (section may be string in model; handlers handle it).

---

**Schedules**

Base route: `/schedule`

- `GET /schedule` — list schedules (protected)
- `POST /schedule` — create (protected)
- `PATCH /schedule/:id` — update (protected)
- `DELETE /schedule/:id` — delete (protected)

Schedule fields: `id, movie_id, theater_id, start_time (YYYY-MM-DD HH:MM:SS), price`.

---

**Tickets / Bookings**

- `POST /tickets/book` — book tickets (protected)
  - Request supports flexible seat formats: either `[{"seat_number":"A1"}]` or `["A1","A2"]` — handler normalizes to seat strings.
  - Requires `schedule_id` and payment info (implementation-specific). Returns booking/transaction info.

- `GET /tickets/my-bookings` — list bookings of the authenticated user

---

**Errors & Status Codes**

- `400 Bad Request` — validation or malformed payload.
- `401 Unauthorized` — missing or invalid access token (protected endpoints).
- `403 Forbidden` — insufficient privileges (admin endpoints).
- `404 Not Found` — resource not found.
- `409 Conflict` — duplicate resource (e.g., movie title already exists).
- `500 Internal Server Error` — unexpected error.

---

**Poster access**

- Posters uploaded via multipart are saved to `public/posters/<filename>` and served at `/posters/<filename>` by the server. The API returns a full URL built from `c.Request.Host` and the `/posters/` path.

If you host the API behind a reverse proxy or at a different base URL, you may want to configure an explicit `APP_BASE_URL` in config and return URLs built from that base.

---

**Environment & Configuration**

- Example keys in `.env.example` (or env vars):
  - `DB_USER`, `DB_PASS`, `DB_HOST`, `DB_PORT`, `DB_NAME` — MySQL connection
  - `APP_PORT` — server port (default in `cmd/api/main.go`)

---

**Database & Migrations**

- The server runs `db.AutoMigrate(...)` at startup for models located in `internal/model` and seeds initial data via `internal/seed/seed.go`.
- If your DB already contains conflicting data (e.g. duplicate movie titles) creating a unique index may fail — clean duplicates first or adjust migrations.

---

**Development Notes & Tests**

- Build:
  ```bash
  go build ./cmd/api
  ./api
  ```
- Run with live reload (recommended): use `air` or similar tools.

---

If you'd like, I can:
- Add OpenAPI / Swagger documentation for the whole API.
- Add example Postman collection.
- Add Link headers for paginated endpoints.

If any endpoint behavior or field naming in this README doesn't match your expected contract, paste an example request/response and I'll update the docs accordingly.

End of README

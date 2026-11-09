# Kwara Viewer

Clipboard of Kwara lots stored in SQLite. Filter by title, city, neighborhood, pickup state and open/ended; sort columns and page 10 rows at a time; refresh from the live `/busca` RSC search.

Host: `https://kwara-viewer.apps.gestaobem.com` (noindex). Public GitHub: [`puppe1990/kwara-viewer-go`](https://github.com/puppe1990/kwara-viewer-go) (the Vite CSV viewer already uses `puppe1990/kwara-viewer`). Open Graph card: `/static/og.png` (1200×630, source `web/static/og.html`).

## Stack

- Go + [Amarra](https://github.com/puppe1990/amarra-cais) + SQLite
- Compact lot shape matches `ferramentas/kwara-mcp` (`id`, `kwaraId`, bid in cents, `pickupState`, `pickupCity`, `pickupAddress`, `endsAt`). City comes from `listing.location`; venue from `pickUpAddress` on the lot ficha.
- Tests use `pkg/cais/fakedata`; live Kwara HTTP is injected

## Commands

```bash
amarra-cais install
amarra-cais test
amarra-cais dev          # http://localhost:8080
cleat__deploy app=kwara-viewer ref=main
```

`POST /sync` with `q=monitor` and `pickup_state=SP` paginates the site search and upserts rows. First boot loads `internal/db/seed-lots.json` when the table is empty.

## Layout

| Path | Role |
| --- | --- |
| `internal/store/` | SQLite lots table, upsert/list/filter |
| `internal/kwara/` | compact mapper + RSC client |
| `internal/handlers/` | catalog HTML, sync, robots |
| `internal/db/seed-lots.json` | SP monitor snapshot |

Do not mix files with `clientes/KwaraBot` or `ferramentas/kwara-mcp`.

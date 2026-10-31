# Flatmate client

React + Vite + TypeScript SPA for the Flatmate Matching Platform.
The Go server exposes the JSON API under `/api/v1` (see
`internal/web/api.go`); this app is the only consumer.

## Prerequisites

- Node 22+
- The Go server running (`make demo` from the repo root serves `:8080`)

## Run

```sh
npm install
npm run dev     # http://localhost:5173, /api proxied to :8080
npm run lint    # tsc + eslint, must be clean
npm run build   # lint + vite build into dist/
```

Production: `make client-build`, then run the server with `-ui=spa`
(or `FLATMATE_UI=spa`); Go serves `client/dist` itself, so the demo
stays one process.

## Notes

- Session is the HttpOnly cookie; CSRF goes in the `X-CSRF-Token`
  header after one `GET /api/v1/csrf` at boot.
- Pinned to `react-router-dom` v7 (`^8.0.0` is not on the npm
  registry under that name). The routing API used here
  (`BrowserRouter`, `Routes`, `Link`, `Navigate`, `Outlet`, hooks)
  is stable across v6/v7, so a future bump is safe.

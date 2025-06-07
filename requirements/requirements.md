# Review‑Board HTTP Service – Requirements Specification

**Version:** 0.1  **Date:** 2025‑06‑07

---

## 1  Purpose

Provide a lightweight, in‑memory micro‑service that stores *review items* and exposes a small REST API so that humans (via curl/Postman) **and** OpenHands (via its SSE tool mechanism) can create, retrieve, update, or delete these items.
The main aim is to make large software packages feasible with openhands by chuncking the actions. 


It asumes documents/code files, it explains the wrongdoing, and proposes a fix. 

## 2  Scope

The service is single‑binary, stateless apart from process RAM, and aims at local/CI use. Durability, clustering, or authentication are **out of scope** for v0.1 but may be added later.

## 3  Stakeholders

| Role                  | Interest                                            |
| --------------------- | --------------------------------------------------- |
| *Developers*          | Maintain, extend, and test the Go code‑base.        |
| *DevOps*              | Package and run the service (Docker, systemd).      |
| *Tooling* (OpenHands) | Discover and call the HTTP + SSE endpoints.         |
| *End‑users*           | Interact via browser or CLI to manage review items. |

## 4  Glossary

| Term            | Meaning                                                                      |
| --------------- | ---------------------------------------------------------------------------- |
| **List**        | Named collection of review items (e.g. `todo`).                              |
| **Item**        | JSON object with `index`, `Document`, `conflict`, `new_statement`, `status`. |
| **Open item**   | Item whose `status == "open"`.                                               |
| **Closed item** | Item whose `status == "closed"`.                                             |

## 5  Functional Requirements

### 5.1 List Management

* **FR‑1** ‑ Create empty list → `GET /add/{list}`.
* **FR‑2** ‑ Create list and seed items → `POST /add/{list}` with body *array <Item>*.
* **FR‑3** ‑ Delete list → `GET /delete/{list}`.
* **FR‑4** ‑ Fetch full list → `GET /list/{list}`.

### 5.2 Item Operations

* **FR‑5** ‑ Fetch first open item → `GET /open/{list}`.
* **FR‑6** ‑ Close item → `GET /close/{list}?index=n`; if `index` missing → close first open item.
* **FR‑7** ‑ Lists auto‑assign 1‑based sequential `index` on creation.
* **FR‑8** ‑ If `status` omitted in payload, default to `"open"`.

### 5.3 Error Handling & Help

* **FR‑9** ‑ Any unknown route or wrong HTTP verb returns **HTTP 400** and a compact usage message (see §7.2).
* **FR‑10** ‑ 404 on missing list; 409 when trying to create an already‑existing list.

### 5.4 OpenHands Integration

* **FR‑11** ‑ Expose SSE stream at `/mcp` and JSON‑RPC back channel at `/mcp/message`.
* **FR‑12** ‑ Advertise five MCP tools (`list_items`, `get_item`, `add_item`, `open_item`, `close_item`) mapped onto the HTTP handlers.

## 6  Non‑Functional Requirements

| ID        | Requirement                                                                |
| --------- | -------------------------------------------------------------------------- |
| **NFR‑1** | Start‑to‑first‑response ≤ 50 ms on a modern laptop.                        |
| **NFR‑2** | Handle 100 concurrent connections without data races (use `sync.RWMutex`). |
| **NFR‑3** | Build with **Go 1.22 +** and only std‑lib plus `mark3labs/mcp-go`.         |
| **NFR‑4** | Default port `8080`; override via `PORT` env var.                          |
| **NFR‑5** | All public responses `Content‑Type: application/json; charset=utf‑8`.      |
| **NFR‑6** | Codebase covered by ≥ 80 % unit tests.                                     |

## 7  API Specification

### 7.1 Item JSON Schema (partial)

```jsonc
type Item = {
  index:        int;   // auto‑assigned (≥1)
  Document:     string;
  conflict:     string;
  new_statement:string;
  status:       "open" | "closed";
}
```

### 7.2 Endpoints

| # | Verb | Path                  | Req. body | Success (200/201) | Error codes   |
| - | ---- | --------------------- | --------- | ----------------- | ------------- |
| 1 | GET  | /add/{list}           | –         | `{message}`       | 409, 400      |
| 2 | POST | /add/{list}           | `[Item]`  | `[Item]`          | 409, 400      |
| 3 | GET  | /delete/{list}        | –         | `{message}`       | 404           |
| 4 | GET  | /list/{list}          | –         | `[Item]`          | 404           |
| 5 | GET  | /open/{list}          | –         | `Item`            | 404, 404‑open |
| 6 | GET  | /close/{list}?index=n | –         | `Item`            | 404, 400      |

Notes:

* **404‑open** – no open item found.
* All error payloads: `{error: string}`.

## 8  Environment & Deployment

* Single binary (`go build -o review‑board`).
* Run directly (`./review‑board`) or via container (`Dockerfile` TBD).
* Optionally fronted by Nginx/Caddy for TLS termination.

## 9  Testing & CI

* Go unit tests located next to implementation with `_test.go` suffix.
* GitHub Actions workflow runs `go vet`, `go test ‑cover`, and builds binary.

## 10  Out of Scope

* Persistent storage beyond RAM.
* AuthN/AuthZ, HTTPS, CORS.
* Pagination for large lists.

## 11  Future Enhancements (non‑blocking)

1. JSON persistence (flush to disk) with configurable period.
2. configurable timeout between serving list items in order to slow down openhands (it makes that the free gemini can run for hours).
3. A page to add jsons to the list and view current lists. 
4. Role‑based access control.
5. Metrics endpoint (`/metrics`, Prometheus).

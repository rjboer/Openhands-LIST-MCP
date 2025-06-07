# OpenHands MCP List Tool
## or coloqually Review-Board HTTP Service

A tiny Go micro-service that stores review items in memory and exposes a handful
of REST-style endpoints.  
It is designed to be called directly by humans (curl/Postman) **or** plugged into
OpenHands as an SSE-tool backend.

---

## Features

| Verb | Path | Purpose |
|------|------|---------|
| **GET**  | `/open/{list}`              | Return **one** open item (first with `status="open"`). |
| **GET**  | `/close/{list}` `?index=n`  | Close item *n* (or the first open one) and return it. |
| **GET**  | `/add/{list}`               | Create an empty list. |
| **POST** | `/add/{list}`               | Create list and seed it with the JSON array in the body. |
| **GET**  | `/delete/{list}`            | Delete the list. |
| **GET**  | `/list/{list}`              | Return the full list as JSON. |

On any unknown route or wrong HTTP verb the service replies **400** plus a
concise usage cheat-sheet.

---

## Quick start

```bash
# clone & build
git clone https://github.com/your-org/review-board.git
cd review-board
go run .

```
![image](https://github.com/user-attachments/assets/aef7325d-34a5-4aa1-a5ba-84b8a7948a50)

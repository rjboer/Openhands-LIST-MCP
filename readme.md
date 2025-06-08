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

## How to use
Clone the git repository

```bash
#Step 1 (dont forget the .)
docker build -t openhands-list-mcp:dev .
#Step 2, (launch it as a deamon -d or without)
docker compose up -d
```
Then Step 3, couple it to openhands, 

Register it in openhands;
I use port 3001 in my docker-compose and I use multiple dockers.. by itself the application hosts on port 8080 (modify via docker file and docker-compose file). 
bridge via host.internal like below. 

![image](https://github.com/user-attachments/assets/ca0121f9-3c5e-4fa4-a42c-9c3951aa906b)



## Impression on port 8080 or 3001
![image](https://github.com/user-attachments/assets/aef7325d-34a5-4aa1-a5ba-84b8a7948a50)


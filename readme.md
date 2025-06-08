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
I use port 3002 in my docker-compose and I use multiple dockers.. by itself the application hosts on port 3002 default (modify via docker file).
The port is modified by the PORT environment variable.  
bridge via host.internal like below (yes it shows an old port, 3001). 

![image](https://github.com/user-attachments/assets/ca0121f9-3c5e-4fa4-a42c-9c3951aa906b)

Step 4: Debugging...

Assuming you run two seperate containers (comes in handy and you dont have to reboot the entire stack every time). 
- First curl from inside your openhands container:
openhands-list-mcp is the hostname of my service..
Try this from inside the openhands docker (depending on your port)

```bash
curl http://openhands-list-mcp:3002
curl http://openhands-list-mcp:8080
curl http://openhands-list-mcp:3002/mcp/health
```
Failure looks like this:
```bash
curl: (6) Could not resolve host: openhands-list-mcp
```
Succes on the /mcp/health handle looks like this:
```bash
Valid endpoints (all JSON):

GET  /open/{list}              → first open item with its index
GET  /close/{list}?index=n     → close item (index optional)
GET  /add/{list}               → create empty list
POST /add/{list}               → create list, seed JSON array
GET  /delete/{list}            → delete list
GET  /list/{list}              → full list JSON
GET  /timeout/{seconds}        → set throttle delay (0-600 s)
GET  /meta                     → summary for index page
/ or /index.html               → web UI
# 
```


The MCP server will tell you in the logs where it is hosted:
Like this:
```bash
[+] Running 1/1
 ✔ Container openhands-list-mcp  Created                                                                                                                      0.0s
Attaching to openhands-list-mcp
openhands-list-mcp  | 🔗  Listening at http://localhost:3002  – UI on /
```


Alternatively, if openhands give an error because it cannot stream....
You should check the routing.....
```bash
2025-06-08 20:56:30.703 | - openhands:ERROR: utils.py:100 - Failed to connect to url='http://host.docker.internal:3001' api_key='******': 
2025-06-08 20:56:30.703 | Traceback (most recent call last):
2025-06-08 20:56:30.703 |   File "/usr/local/lib/python3.12/asyncio/tasks.py", line 520, in wait_for
2025-06-08 20:56:30.703 |     return await fut
2025-06-08 20:56:30.703 |            ^^^^^^^^^
2025-06-08 20:56:30.703 |   File "/app/openhands/mcp/client.py", line 71, in connect_with_timeout
2025-06-08 20:56:30.703 |     streams = await self.exit_stack.enter_async_context(streams_context)
2025-06-08 20:56:30.703 |               ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

On linux one might create a specific network (in Windows you can use host.docker.internal, but it might not work in Linux). 
```bash
docker network create openhands-net
```





## Impression on port 8080 or 3001
![image](https://github.com/user-attachments/assets/aef7325d-34a5-4aa1-a5ba-84b8a7948a50)


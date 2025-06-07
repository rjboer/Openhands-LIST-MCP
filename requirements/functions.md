
# Review-Board HTTP Service - Function Plan

## 0. Application Initialization

0.  **Function 0: `main`**
    *   Description: Initializes the application, sets up the HTTP server, and handles graceful shutdown.
    *   Inputs: None.
    *   Outputs: None.

## 1. HTTP Handlers

1.  **Function 1: `GET /open/{list}`**
    *   Description: Returns one open item from the list.
    *   Inputs: `list` (string, path parameter).
    *   Outputs: JSON (item), `error`.

2.  **Function 2: `GET /close/{list}`**
    *   Description: Closes an item in the list and returns it.
    *   Inputs: `list` (string, path parameter), `index` (integer, query parameter, optional).
    *   Outputs: JSON (item), `error`.

3.  **Function 3: `GET /add/{list}`**
    *   Description: Creates an empty list.
    *   Inputs: `list` (string, path parameter).
    *   Outputs: None, `error`.

4.  **Function 4: `POST /add/{list}`**
    *   Description: Creates a list and seeds it with items from the request body.
    *   Inputs: `list` (string, path parameter), `body` (JSON array).
    *   Outputs: None, `error`.

5.  **Function 5: `GET /delete/{list}`**
    *   Description: Deletes a list.
    *   Inputs: `list` (string, path parameter).
    *   Outputs: None, `error`.

6.  **Function 6: `GET /list/{list}`**
    *   Description: Returns the full list as JSON.
    *   Inputs: `list` (string, path parameter).
    *   Outputs: JSON (list), `error`.


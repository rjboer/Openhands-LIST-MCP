# ---------- build stage ----------
FROM golang:1.22 AS builder
WORKDIR /src

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 PORT=3002

COPY go.mod go.sum ./
RUN go mod download

# main.go is in repo root, so copy *everything* then build “.”
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /tmp/openhands-list-mcp .

# ---------- final image ----------
FROM scratch
COPY --from=builder /tmp/openhands-list-mcp /app
ENTRYPOINT ["/app"]

FROM golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && go build -ldflags="${LDFLAGS}" -o /out/mcp-server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/mcp-server /app/mcp-server
EXPOSE 8081
ENTRYPOINT ["/app/mcp-server"]

FROM golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY proto /src/proto
COPY agent-server /src/agent-server
WORKDIR /src/agent-server
RUN go build -ldflags="${LDFLAGS}" -o /out/agent-server ./cmd/server

FROM alpine:3.20
COPY --from=build /out/agent-server /usr/local/bin/agent-server
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/agent-server"]

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY proto /src/proto
COPY mcp-server /src/mcp-server
WORKDIR /src/mcp-server
RUN go env -w GOPROXY=https://goproxy.cn,direct && go build -ldflags="${LDFLAGS}" -o /out/mcp-server ./cmd/server

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
WORKDIR /app
COPY --from=build /out/mcp-server /app/mcp-server
EXPOSE 8081
ENTRYPOINT ["/app/mcp-server"]

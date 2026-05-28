FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY proto /src/proto
COPY backend /src/backend
WORKDIR /src/backend
RUN go env -w GOPROXY=https://goproxy.cn,direct && go build -ldflags="${LDFLAGS}" -o /out/backend-api ./cmd/api

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
WORKDIR /app
COPY --from=build /out/backend-api /app/backend-api
EXPOSE 8080 8082
ENTRYPOINT ["/app/backend-api"]

FROM golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && go build -ldflags="${LDFLAGS}" -o /out/backend-api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/backend-api /app/backend-api
EXPOSE 8080 8082
ENTRYPOINT ["/app/backend-api"]

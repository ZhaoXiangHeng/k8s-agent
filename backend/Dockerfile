FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && go build -o /out/backend-api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/backend-api /app/backend-api
EXPOSE 8080
ENTRYPOINT ["/app/backend-api"]

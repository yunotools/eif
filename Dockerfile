FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/eif ./cmd/eif

FROM alpine:3.21
RUN addgroup -S eif && adduser -S -G eif eif
WORKDIR /app
COPY --from=build /out/eif /app/eif
COPY web/static /app/web/static
USER eif
EXPOSE 8080
ENTRYPOINT ["/app/eif"]

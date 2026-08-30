FROM golang:1.26-alpine AS build
WORKDIR /yuno
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/eif ./cmd/eif

FROM alpine:3.21
RUN addgroup -S eif && adduser -S -G eif eif
WORKDIR /yuno
COPY --from=build /out/eif /yuno/eif
COPY web/static /yuno/web/static
RUN mkdir -p /yuno/runtime && chown -R eif:eif /yuno/runtime
USER eif
EXPOSE 8080
ENTRYPOINT ["/yuno/eif"]

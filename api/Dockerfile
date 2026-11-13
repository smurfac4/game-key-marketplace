FROM golang:1.26-alpine AS build
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/ ./cmd/server ./cmd/migrate ./cmd/seed

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/ /usr/local/bin/
EXPOSE 8080
CMD ["server"]

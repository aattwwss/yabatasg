FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Pre-compile heavy deps into the Go build cache. This layer is
# Docker-cached when go.mod/go.sum don't change, cutting the final
# build step from ~10s to ~3s.
RUN go build modernc.org/sqlite

COPY . .
RUN go build -ldflags="-s -w" -o /yabatasg .

FROM alpine:3.21

RUN mkdir -p /data

COPY --from=build /yabatasg /yabatasg

EXPOSE 8080

CMD ["/yabatasg"]

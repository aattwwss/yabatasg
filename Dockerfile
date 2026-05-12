FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Pre-compile the heavy sqlite dependency into the Go build cache.
# This layer stays cached as long as go.mod/go.sum don't change,
# so Docker rebuilds skip the 30-60s sqlite compilation.
RUN go build modernc.org/sqlite

COPY . .
RUN go build -o /yabatasg .

FROM alpine:3.21

RUN mkdir -p /data

COPY --from=build /yabatasg /yabatasg

EXPOSE 8080

CMD ["/yabatasg"]

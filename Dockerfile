FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /yabatasg .

FROM alpine:3.21

ARG UID=1000
RUN adduser -D -H -u ${UID} appuser && mkdir -p /data && chown appuser /data

COPY --from=build /yabatasg /yabatasg

EXPOSE 8080

USER appuser

CMD ["/yabatasg"]

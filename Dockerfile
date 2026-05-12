FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /yabatasg .

FROM alpine:3.21

RUN mkdir -p /data

COPY --from=build /yabatasg /yabatasg

EXPOSE 8080

CMD ["/yabatasg"]

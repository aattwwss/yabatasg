FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /yabatasg main.go

#EXPOSE 8080

CMD [ "/yabatasg" ]

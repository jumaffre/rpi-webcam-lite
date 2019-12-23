FROM golang:latest

LABEL maintainer="Julien Maffre <jumaffre@microsoft.com>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 

COPY . .

RUN go build -o main . 

EXPOSE 4443

CMD ["./projecta"]

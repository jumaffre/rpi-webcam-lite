FROM golang:latest

LABEL maintainer="Julien Maffre <maffre.jul@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 

COPY . .

RUN go build -o projecta . 

EXPOSE 4443

CMD ["./projecta"]

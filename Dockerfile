# First, build app
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 
COPY . .
RUN go build -o projecta . 


# Then, create minimal image for app
# TODO: Use scratch image?
FROM golang:alpine

COPY --from=builder /app/projecta /app/projecta

# For the app
EXPOSE 4443 
# For Let's Encrypt
EXPOSE 4444

ENTRYPOINT ["/app/projecta"]

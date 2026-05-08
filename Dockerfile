FROM golang:1.24-alpine

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-w -s" -o throttle .

EXPOSE 8080

CMD ["./throttle"]
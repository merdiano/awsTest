FROM golang:1.19 AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=build /app/app /app

WORKDIR /app

CMD ["./app"]
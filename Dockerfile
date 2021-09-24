FROM golang:1.17 as compiler
RUN mkdir /builder
WORKDIR /builder

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

COPY --from=compiler /builder/main ./

CMD ["./main"]

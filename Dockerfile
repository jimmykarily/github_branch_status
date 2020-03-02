FROM golang:1.14 AS builder

RUN mkdir /src
WORKDIR /src
ADD main.go .
RUN go build -o server -ldflags "-linkmode external -extldflags -static" -a main.go

FROM scratch
COPY --from=builder /src/server /server
CMD ["/server"]

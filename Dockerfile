FROM golang:1.14 AS builder

RUN mkdir /src
WORKDIR /src
ADD main.go .
RUN go build -o server -ldflags "-linkmode external -extldflags -static" -a main.go

FROM opensuse/leap:15.2
COPY --from=builder /src/server /server
COPY images/ .
CMD ["/server"]

FROM golang:1.17 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /application

##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /
COPY --from=builder /application /application
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/application"]


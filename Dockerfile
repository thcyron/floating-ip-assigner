FROM golang:1.11 as build
COPY . /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -o assigner main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=build /src/assigner /assigner
ENTRYPOINT ["/assigner"]

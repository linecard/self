FROM golang:alpine3.19 as build
WORKDIR /src
COPY ./src .

RUN go mod download
RUN GOOS=linux CGO_ENABLED=0 go build -o main

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --chmod=755 ./bin/docker /bin/docker
COPY --chmod=+x --from=build /src/main /var/task/main

ENTRYPOINT ["/var/task/main"]
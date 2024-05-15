FROM --platform=$BUILDPLATFORM golang:alpine3.18 as build
WORKDIR /src
COPY ./src .
RUN go mod download
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o main

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=+x --from=build /src/main /var/task/main
ENTRYPOINT [ "/var/task/main" ]

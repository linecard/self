FROM golang:alpine3.20 as build
WORKDIR /src
COPY main.go .
RUN GOOS=linux CGO_ENABLED=0 go build main.go

FROM scratch
COPY --chmod=+x --from=build /src/main /var/task/main
ENTRYPOINT [ "/var/task/main" ]
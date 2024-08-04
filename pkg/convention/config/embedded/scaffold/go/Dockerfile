FROM golang:alpine3.20 as build
WORKDIR /src
COPY main.go .
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN GOOS=linux CGO_ENABLED=0 go build main.go

FROM scratch
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
COPY --chmod=+x --from=build /src/main /var/task/main

ENV AWS_LWA_PORT=8081
ENTRYPOINT [ "/var/task/main" ]
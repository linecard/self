FROM golang:alpine3.20 as build
RUN go get -u github.com/gin-gonic/gin
WORKDIR /src
COPY main.go .
RUN GOOS=linux CGO_ENABLED=0 go build main.go

FROM scratch
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

COPY --chmod=+x --from=build /src/main /var/task/main

ENTRYPOINT [ "/var/task/main" ]
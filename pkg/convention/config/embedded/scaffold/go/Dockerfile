FROM golang:alpine3.20 as build
WORKDIR /src
COPY main.go .
RUN GOOS=linux CGO_ENABLED=0 go build main.go

FROM scratch
COPY --chmod=+x --from=build /src/main /var/task/main

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.3 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

ENTRYPOINT [ "/var/task/main" ]
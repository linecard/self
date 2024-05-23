FROM python:alpine3.20
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.3 /lambda-adapter /opt/extensions/lambda-adapter
COPY ./main.py /var/task/main.py
ENTRYPOINT [ "python", "/var/task/main.py" ]

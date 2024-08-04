FROM python:alpine3.20

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter

WORKDIR /var/task
RUN pip install fastapi uvicorn
COPY main.py .

ENV AWS_LWA_PORT=8081
ENTRYPOINT [ "uvicorn", "main:app" ]

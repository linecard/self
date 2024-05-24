FROM python:alpine3.20

RUN pip install fastapi uvicorn
COPY ./main.py /var/task/main.py

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.3 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

ENTRYPOINT [ "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080"]

FROM ruby:3.2.5

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

RUN gem install sinatra --no-document

WORKDIR /var/task
COPY main.py .

ENTRYPOINT [ "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080"]

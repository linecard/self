FROM ruby:3.2.5

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter

WORKDIR /var/task
RUN gem install sinatra rackup --no-document
COPY main.rb .

ENV AWS_LWA_PORT=8081
ENTRYPOINT [ "ruby", "main.rb" ]

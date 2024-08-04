FROM ruby:3.2.5

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

WORKDIR /var/task
RUN gem install sinatra rackup --no-document
COPY main.rb .

ENTRYPOINT [ "ruby", "main.rb" ]

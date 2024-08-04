FROM node:lts-alpine

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter

WORKDIR /var/task
RUN npm install express
COPY main.js .

ENV AWS_LWA_PORT=8081
ENTRYPOINT [ "node", "main.js" ]
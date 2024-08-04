FROM node
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.4 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

RUN npm install express

WORKDIR /var/task
COPY main.js .

ENTRYPOINT [ "node", "main.js" ]
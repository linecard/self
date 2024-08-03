FROM node
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.3 /lambda-adapter /opt/extensions/lambda-adapter
ENV AWS_LWA_PORT=8080

COPY index.js /var/task/index.js

ENTRYPOINT [ "node", "var/task/index.js" ]
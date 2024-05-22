FROM python:3.11 as runtime
ENV LAMBDA_TASK_ROOT=/var/task
WORKDIR ${LAMBDA_TASK_ROOT}
RUN pip install awslambdaric
ENTRYPOINT [ "/usr/local/bin/python", "-m", "awslambdaric" ]

FROM runtime
COPY src/ ${LAMBDA_TASK_ROOT}
CMD ["main.handler"]
FROM scratch

COPY --from=ghcr.io/linecard/self:latest /ko-app/self /var/task/self

ENTRYPOINT ["/var/task/self"]
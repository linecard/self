FROM scratch

COPY --from=ghcr.io/linecard/self:{{ .Version }}/ko-app/self /var/task/self

ENTRYPOINT ["/var/task/self"]
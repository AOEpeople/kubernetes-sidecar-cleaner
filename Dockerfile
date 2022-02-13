FROM scratch
COPY app /sidecar-cleanup

CMD ["/sidecar-cleanup"]

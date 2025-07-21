FROM vmi-base

USER root
WORKDIR /usr/local/vmi-infra

COPY vmi-infra/files/base .
RUN \
    victoria-metrics/download-victoria-metrics.sh && \
    victoria-metrics/create-victoria-metrics-runtime-symlinks.sh
RUN \
    grafana/download-grafana.sh && \
    grafana/create-grafana-runtime-symlinks.sh

COPY vmi-infra/files/update/ .

USER ${VMI_USER}
CMD ["./run-vmi-infra.sh"]

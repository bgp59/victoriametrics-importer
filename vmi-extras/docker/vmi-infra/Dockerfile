FROM vmi-base

USER root
WORKDIR /usr/local/vmi-infra

# Install base software only:
COPY --exclude=.DS_Store vmi-infra/install install
RUN \
    install/install-vmi-infra.sh -r . -R /volumes/runtime/vmi-infra -b \
    && chown ${VMI_USER}:${VMI_USER} -R /volumes/runtime/vmi-infra

# Apply updates starting with the least likely to change:
COPY --exclude=.DS_Store vmi-infra/update/scripts scripts
COPY --exclude=.DS_Store vmi-infra/update/victoria-metrics victoria-metrics
RUN chown ${VMI_USER}:${VMI_USER} -R victoria-metrics/auth victoria-metrics/tls
COPY --exclude=.DS_Store vmi-infra/update/grafana grafana

USER ${VMI_USER}
CMD ["./scripts/run-vmi-infra"]

FROM ubuntu:20.04

ARG TARGETARCH

RUN \
    apt update \
    && ln -fs /usr/share/zoneinfo/GMT /etc/localtime \
        && apt-get install -y tzdata \
    && (yes | unminimize) \
    && apt-get install -y \
        curl \
        gcc \
        dnsutils \
        iputils-ping \
        iproute2 \
        libcap2-bin \
        man-db \
        rsync \
        sudo \
        strace

RUN \
    apt-get install -y \
        python3.9-full \
        python3-pip \
    && pip install -U pip

ENV TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-${TARGETARCH} /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]


ENV VMI_USER=vmi

RUN \
    useradd -U -d /home/${VMI_USER} -m -s /bin/bash ${VMI_USER} && \
    usermod -aG sudo ${VMI_USER} && \
    mkdir -p /volumes/runtime && \
    chown ${VMI_USER}:${VMI_USER} /volumes/runtime && \
    echo "vmi ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/vmi && \
    chmod 0440 /etc/sudoers.d/vmi

VOLUME /volumes/runtime

WORKDIR /home/${VMI_USER}
USER ${VMI_USER}
CMD ["sleep", "infinity"]

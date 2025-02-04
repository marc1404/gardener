FROM kindest/node:v1.32.1@sha256:6afef2b7f69d627ea7bf27ee6696b6868d18e03bf98167c420df486da4662db6

ARG TARGETOS
ARG TARGETARCH

RUN apt-get update -yq && \
    apt-get install -yq --no-install-recommends wget apparmor apparmor-utils jq openssh-server sudo

# remove kind's kubelet unit
RUN rm -f /etc/systemd/system/kubelet.service && \
    rm -rf /etc/systemd/system/kubelet.service.d/

# copy containerd hosts configurations for local registry mirrors
COPY hosts.toml /etc/containerd/certs.d/garden.local.gardener.cloud:5001/hosts.toml

# add our userdata executor unit
COPY run-userdata.sh /run-userdata.sh
COPY run-userdata.service /etc/systemd/system
RUN systemctl enable run-userdata.service

ENTRYPOINT ["/usr/local/bin/entrypoint", "/sbin/init"]

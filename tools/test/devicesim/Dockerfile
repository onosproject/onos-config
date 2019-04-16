FROM debian:stretch
MAINTAINER Sean Condon <sean@opennetworking.org>
LABEL description="Builds a gNMI simulator on a Debian distribution"

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qy --no-install-recommends \
      apt-utils \
      ca-certificates \
      software-properties-common \
    && echo "deb http://deb.debian.org/debian stretch-backports main" >  /etc/apt/sources.list.d/backports.list \
    && apt-get update

RUN DEBIAN_FRONTEND=noninteractive apt-get install -qy --no-install-recommends \
    fping \
    git \
    golang-1.11-go \
    iproute2 \
    iputils-ping \
    net-tools \
    netcat-openbsd \
    openssh-client \
    psmisc \
    sudo \
    vim

ENV HOME=/home/devicesim
RUN mkdir $HOME
WORKDIR $HOME

ENV GOROOT=/usr/lib/go-1.11
ENV GOBIN=$GOROOT/bin
ENV PATH=$GOBIN:${PATH}
ENV GOPATH=$HOME/go

ENV INDEX=1

RUN mkdir -p $GOPATH \
    && go get -u \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/google/gnxi/gnmi_target

RUN go install -v \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/google/gnxi/gnmi_target

ENV GNMI_TARGET=localhost
ENV GNMI_PORT=10161

COPY ./target_configs target_configs
COPY ./certs certs

CMD certs/generate_certs.sh device $INDEX > /dev/null 2>&1 && \
  IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
  echo "Please add '"$IPADDR" device"$INDEX" device"$INDEX".opennetworking.org' to /etc/hosts and access with gNMI client at device"$INDEX":"$GNMI_PORT && \
  sed -i -e "s/replace-device-name/device"$INDEX"/g" target_configs/typical_ofsw_config.json && \
  gnmi_target \
  -bind_address :$GNMI_PORT \
  -key $HOME/certs/device$INDEX.key \
  -cert $HOME/certs/device$INDEX.crt \
  -ca $HOME/certs/onfca.crt \
  -alsologtostderr \
  -config target_configs/typical_ofsw_config.json > /dev/null 2>&1



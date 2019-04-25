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

RUN mkdir -p $GOPATH \
    && go get -u \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/google/gnxi/gnmi_target \
      github.com/openconfig/gnmi/cmd/gnmi_cli
      

RUN go install -v \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/google/gnxi/gnmi_target \
      github.com/openconfig/gnmi/cmd/gnmi_cli

ENV GNMI_TARGET=localhost
ENV GNMI_PORT=10161

COPY ./target_configs target_configs
COPY ./certs certs

CMD \
  IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
  if [ "$GNMI_TARGET" != "localhost" ]; then \
      certs/generate_certs.sh $GNMI_TARGET > /dev/null 2>&1; \
      echo "Please add '"$IPADDR" "$GNMI_TARGET"' to /etc/hosts and access with gNMI client at "$GNMI_TARGET":"$GNMI_PORT; \
  else \
      echo "gNMI running on localhost:"$GNMI_PORT; \
  fi && \
  sed -i -e "s/replace-device-name/"$GNMI_TARGET"/g" target_configs/typical_ofsw_config.json && \
  sed -i -e "s/replace-motd-banner/Welcome to gNMI service on "$GNMI_TARGET":"$GNMI_PORT"/g" target_configs/typical_ofsw_config.json && \
  gnmi_target \
  -bind_address :$GNMI_PORT \
  -key $HOME/certs/$GNMI_TARGET.key \
  -cert $HOME/certs/$GNMI_TARGET.crt \
  -ca $HOME/certs/onfca.crt \
  -alsologtostderr \
  -config target_configs/typical_ofsw_config.json > /dev/null 2>&1



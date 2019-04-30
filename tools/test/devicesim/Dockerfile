FROM debian:stretch
MAINTAINER Sean Condon <sean@opennetworking.org>, Adib Rastegarnia <adib.rastegarnia@gmail.com>
LABEL description="Builds a gNMI/gNOI simulator on a Debian distribution"

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
      github.com/openconfig/gnmi/cmd/gnmi_cli \
      github.com/google/gnxi/gnoi_target \ 
      github.com/google/gnxi/gnoi_cert 
      

RUN go install -v \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/google/gnxi/gnmi_target \
      github.com/openconfig/gnmi/cmd/gnmi_cli \
      github.com/google/gnxi/gnoi_target \ 
      github.com/google/gnxi/gnoi_cert 

ENV GNMI_PORT=10161
ENV HOST_TARGET=localhost
ENV GNOI_PORT=50001
ENV SIM_MODE=1

COPY ./target_configs target_configs
COPY ./certs certs
COPY run_targets.sh .


CMD \
  chmod +x run_targets.sh && ./run_targets.sh 

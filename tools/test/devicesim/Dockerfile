FROM debian:stretch
LABEL maintainer="Sean Condon <sean@opennetworking.org>, Adib Rastegarnia <adib@opennetworking.org> "
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
    procps \
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
      github.com/openconfig/gnmi/cmd/gnmi_cli \
      github.com/google/gnxi/gnoi_target \ 
      github.com/google/gnxi/gnoi_cert 
      

RUN go install -v \
      github.com/google/gnxi/gnmi_capabilities \
      github.com/google/gnxi/gnmi_get \
      github.com/google/gnxi/gnmi_set \
      github.com/openconfig/gnmi/cmd/gnmi_cli \
      github.com/google/gnxi/gnoi_target \ 
      github.com/google/gnxi/gnoi_cert


ENV ONOS_CONFIG_ROOT=$GOPATH/src/github.com/onosproject/onos-config/
ENV GNMI_PORT=10161
ENV GNOI_PORT=50001
ENV SIM_MODE=1

RUN mkdir -p $ONOS_CONFIG_ROOT/tools/test/devicesim

COPY target_configs target_configs
COPY certs certs
COPY scripts scripts
COPY gnmi_target gnmi_target
COPY gnmi $GOPATH/src/github.com/onosproject/onos-config/tools/test/devicesim/gnmi


RUN cd $GOPATH/src/github.com/onosproject/onos-config/tools/test/devicesim/gnmi && go install
RUN cd ./gnmi_target && go install

RUN chmod +x ./scripts/run_targets.sh

CMD ["./scripts/run_targets.sh"]

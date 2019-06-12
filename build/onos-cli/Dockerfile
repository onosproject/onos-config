ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION as builder
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && make build

FROM alpine:3.8

RUN apk upgrade --update --no-cache && apk add bash bash-completion
RUN addgroup -S onos && adduser -S -G onos onos

USER onos
WORKDIR /home/onos

COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/onos /usr/local/bin/onos

RUN onos init && \
    cp /etc/profile /home/onos/.bashrc && \
    onos completion bash > /home/onos/.onos/bash_completion.sh && \
    onos config set address onos-config-onos-config:5150 && \
    echo "source /home/onos/.onos/bash_completion.sh" >> /home/onos/.bashrc

ENTRYPOINT ["/bin/bash"]
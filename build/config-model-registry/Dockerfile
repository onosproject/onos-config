ARG ONOS_CONFIG_VERSION=latest

FROM onosproject/onos-config-base:${ONOS_CONFIG_VERSION}

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config/cmd/config-model", "registry", "serve"]

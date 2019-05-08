## Building the Config Manager

### Makefile

The config manager provides a `Makefile` that supports all steps required to test
and build the project. The makefile supports the following targets:
* `test` - pulls dependencies and runs tests, including lint and vet
* `build` - runs `test` and builds an executable
* `all` - runs `test`, `build`, and creates the `onosproject/onos-config` Docker image
  * Requires a local Docker daemon
  * Requires the `ONOS_CONFIG_VERSION` environment variable to be set

### Building with the Docker image

The build requires numerous dependencies for linting, vetting, and compiling the
project. These dependencies are conveniently provided in the `onosproject/onos-config-build`
image. To build with the build image, mount the onos-config root to the container
and pass the desired argument to the Makefile:

```
docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/onos-config-build:0.3 build
```

### Running the Docker image in development

To build the Docker image, set the `ONOS_CONFIG_VERSION` environment variable and run `make`:
```
ONOS_CONFIG_VERSION=0.1 make
```

To run the image for development, you must map the ports to the host and mount
test configuration files to the container:

```
docker run -p 8080:8080 -p 5150:5150 -v `pwd`/configs:/etc/onos-config-manager -it onosproject/onos-config:0.1 \
-restconfPort=8080 \
-configStore=/etc/onos-config-manager/configStore-sample.json \
-changeStore=/etc/onos-config-manager/changeStore-sample.json \
-deviceStore=/etc/onos-config-manager/deviceStore-sample.json \
-networkStore=/etc/onos-config-manager/networkStore-sample.json
```

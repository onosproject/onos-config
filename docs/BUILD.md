## Building onos-config

Note: this assumes you have all the [Prerequisites](./Prerequisites.md) steps done. 

### Makefile

The project provides a `Makefile` that supports all steps required to test
and build the project.  
The makefile supports the following targets:
* `test` - pulls dependencies and runs tests, including lint, vet and license
* `build` - runs `test` and builds an executable
* `all` - runs `test`, `build`, and creates the `onosproject/onos-config` Docker image
  * Requires a local Docker daemon


### Building with the Docker image

The build requires numerous dependencies for linting, vetting, and compiling the
project. These dependencies are conveniently provided in the `onosproject/onos-config-build`
image. To build with the build image, mount the onos-config root to the container
and pass the desired argument to the Makefile:

```
docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/onos-config-build:0.3 build
```

### Running the Docker image in development

To build the Docker image run `make` in the `onos-config` folder:
```
make
```

To run the image for development, you must map the ports to the host and mount
test configuration files to the container:

```
docker run -p 5150:5150 -v `pwd`/configs:/etc/onos-config-manager -it onosproject/onos-config:0.1 \
-configStore=/etc/onos-config-manager/configStore-sample.json \
-changeStore=/etc/onos-config-manager/changeStore-sample.json \
-deviceStore=/etc/onos-config-manager/deviceStore-sample.json \
-networkStore=/etc/onos-config-manager/networkStore-sample.json
```

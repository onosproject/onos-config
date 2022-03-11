<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Extending onos-config with Configuration Model Plugins

`onos-config` is an extensible configuration management system, that allows the
configuration of many types and versions of targets (devices) to be managed
concurrently.

Information models in YANG format (RFC 6020) can be used to accurately define
the configuration and state objects and attributes of a device. In practice a
target's object model usually consists of a number of YANG files including
augments and deviations, all of which must be considered as a combined unit.

In `onos-config` a set of these combined YANG files defining a particular
version of a target type is known as a **model**.

Over its lifecycle onos-config will have to deal with different models as
its scope is expanded and as devices - and configurable targets in general -
go through their release cycles. To allow this, models are loadable as plugins in
the form of sidecar containers and are accessible to `onos-config` via a gRPC interface 
that can be used to perform a number of model-specific tasks:

* Retrieve target capabilities (model schema in form of configuration paths)
* Validate target JSON configuration (for adherence to the schema and any consistency rules)
* Extract path/type/value list from target JSON configuration

## Model Compiler

Plugins are built from a set of specified YANG files and a small `metadata.yaml` file,
using facilities of a `onosproject/model-compiler` docker image. This docker image utilizes
a number of off-the-shelf tools such as YGOT, pyang and automatically generates code and 
a Makefile that can be used to build, assemble and publish the model plugin docker image.

### Create your own Model Plugin
To build your own configuration model plugin, assemble all the required YANG files and place
them in the `yang/` directory located at the root of your project. Then, author a
small YAML file `metadata.yaml` at the root of the project. This file names the model,
specifies its version, names the Go package for the generated code and
identifies the root YANG files that should be passed to YGOT by the model compiler.
Here's an example:
```yaml
name: mymodel
version: 1.0.0
artifactName: devicesim
goPackage: github.com/onosproject/config-models/models/mymodel
modules:
  - name: openconfig-interfaces
    organization: OpenConfig working group
    revision: 2017-07-14
    file: openconfig-interfaces@2017-07-14.yang
  - name: openconfig-openflow
    organization: OpenConfig working group
    revision: 2017-06-01
    file: openconfig-openflow@2017-06-01.yang
  - name: openconfig-platform
    organization: OpenConfig working group
    revision: 2016-12-22
    file: openconfig-platform@2016-12-22.yang
  - name: openconfig-system
    organization: OpenConfig working group
    revision: 2017-07-06
    file: openconfig-system@2017-07-06.yang
```
Note that the `yang/` directory will also need to contain all YANG files being included/referenced
by any of the root modules.

## Invoke the model compiler
Once the YANG files and metadata YAML file are ready invoke the model compiler as follows:
```shell
docker run -v $(pwd):/config-model onosproject/model-compiler:latest
```

Afterwards, to compile and assemble the configuration model docker image, simply run:
```shell
make image
```


## Loading the Model Plugin

## Model Plugins and gNMI Capabilities
### Capabilities on gNMI Northbound interface
The CapabilitiesResponse on the gNMI northound interface is generated dynamically
from the `modeldata` section of all of the loaded Model Plugins.

### Capabilities comparison on Southbound gNMI interface
At runtime when devices are connected to onos-config the response to the
Capabilities request are compared with the
modeldata for their corresponding ModelPlugin - if there is not an exact
match a warning is displayed.

## OpenConfig Models
Some devices that support OpenConfig Models report their capabilities using an
OpenConfig versioning scheme e.g. 0.5.0, rather than the YANG revision date in
the format 2017-07-06. If the device can correct its capabilities to give the
revision then it should to be more consistent with non OpenConfig YANG models.

Accessing OpenConfig model of a specific revision requires a number of steps in
[Github](https://github.com/openconfig/public).

For instance if a device reports it used openconfig-interfacess.yang `2.0.0`,
then to get this file do:

* Browse to [openconfig-interfaces.yang](https://github.com/openconfig/public/blob/master/release/models/interfaces/openconfig-interfaces.yang)
* Observe in the list of `revision` items in the YANG file that the reference
`2.0.0` corresponds to a release date of `2017-07-14`
* Click in the `History` button
* In the `History` page for this file, see that the next commit after this
date was on `Aug 9, 2017`
* Click on the related `commit message`
* In the list of files modified in that commit click the `...` next to the file
`openconfig-interfacess.yang` and choose `View File`
* In the page that displays the historical version of the file, click the `Raw` button
* In the resulting raw display of the YANG file verify that the latest revision is `2017-07-14`
* Save the file locally as `openconfig-interfaces@2017-07-14.yang`

All the files in the [yang](../modelplugin/yang) folder were downloaded in this
way. They are not strictly needed once `generated.go` has been created, but are
kept here for convenience, saving to have to run the procedure above if a change
was needed.

> If the generator program reports that a dependency was required e.g.
`openconfig-inet-types.yang` then the version of this file with a date equal
to or before 2017-07-14 should be downloaded - it is `openconfig-inet-types@2017-07-14.yang`

### Readonly paths in OpenConfig models
When an item in an Openconfig YANG file has "config false" it is effectively a
read-only attribute. Usually with OpenConfig read-only objects are interspersed
throughout the YANG model.

To see a list of Read Only paths use the command:
```bash
> onos config get plugins -v
```

When the Model Plugin is loaded, setting of an attribute like `state/address`
should give an appropriate error

```bash
> gnmi_cli -address onos-config:5150 -set \
    -proto "update: <path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'openflow'> elem: <name: 'controllers'> elem: <name: 'controller' key: <key: 'name' value: 'main'>> elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>> elem: <name: 'state'> elem: <name: 'address'>> val: <string_val: '192.0.2.11'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
gives the error:
```bash
rpc error: code = InvalidArgument desc = update contains a change to a read only
  path /system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/address. Rejected
```

## Troubleshooting

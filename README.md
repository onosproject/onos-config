# onos-config
Configuration subsystem for µONOS - a new generation ONOS architecture

## Design Objectives
gNMI provides transactionality with respect to a single device; all configuration operations specified as part of a single batch are all applied or none of them are. The core capability of the configuration platform should build on this gNMI feature to:

* Provide ability to apply a batch of operations (via NB gNMI API) targeted at multiple devices to be performed as a transaction; either all changes are applied to all devices or none of the changes are applied to any of the devices.
* Track configuration change transactions applied to a set of devices over time and allow rollback to previous configuration checkpoints demarcated by the transactions.

The above features are the principal objectives for the configuration management platform. Second order objectives are as follows:

* Support high-availability and distributed operation, including ISSU
* Support networks comprising of ~30 devices, ~2000 ports and rate of ~10K configuration transactions per day

### Additional Features (to be integrated with the above)
The following set of features will be required to support the real-world use-cases, but may not necessarily be part of the code configuration subsystem and instead can be provided as additional layers:

* Ability to preload initial configuration about devices that have not yet been discovered - or even deployed
* Dry run capability - validate configuration with respect to model, but also with respect to existing business constraints or policies
* Import existing device configuration
* Configuration Intents - ability to translate high-level (possibly network-wide) configuration specifications into lower-level (device-specific) operations

## Key Tenets
* The subsystem will be designed as a separate entity to allow its use with the existing ONOS 2.0 architecture and to fit with the NG ONOS architecture.
* Northbound API for the subsystem will be gNMI and gNOI.
   * These interfaces are already accepted standards and are well-defined, low-profile interfaces with support for transaction semantics.
   * YANG models that will be exposed as part of the NB API are yet to be determined (or defined).
* Southbound API for the subsystem will be gNMI and gNOI.
   * This will allow direct use with Stratum-compliant switches without requiring an adaptation layer.
   * Adapters can be created for devices that do not directly support gNMI/gNOI interfaces. Such adapters can be deployed either as proxy agents or potentially can be hosted on the devices themselves.

## Additional Documentation
* [How to contribute][contributing] to µONOS
* [How to run][running] µONOS

[contributing]: https://github.com/opennetworkinglab/onos-config/blob/master/docs/CONTRIBUTING.md
[running]:  https://github.com/opennetworkinglab/onos-config/blob/master/docs/RUN.md

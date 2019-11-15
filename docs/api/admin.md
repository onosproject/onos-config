# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/admin/admin.proto](#api/admin/admin.proto)
    - [Chunk](#onos.config.admin.Chunk)
    - [CompactChangesRequest](#onos.config.admin.CompactChangesRequest)
    - [CompactChangesResponse](#onos.config.admin.CompactChangesResponse)
    - [GetSnapshotRequest](#onos.config.admin.GetSnapshotRequest)
    - [ListModelsRequest](#onos.config.admin.ListModelsRequest)
    - [ListSnapshotsRequest](#onos.config.admin.ListSnapshotsRequest)
    - [ModelInfo](#onos.config.admin.ModelInfo)
    - [ReadOnlyPath](#onos.config.admin.ReadOnlyPath)
    - [ReadOnlySubPath](#onos.config.admin.ReadOnlySubPath)
    - [ReadWritePath](#onos.config.admin.ReadWritePath)
    - [RegisterResponse](#onos.config.admin.RegisterResponse)
    - [RollbackRequest](#onos.config.admin.RollbackRequest)
    - [RollbackResponse](#onos.config.admin.RollbackResponse)
  
    - [Type](#onos.config.admin.Type)
  
  
    - [ConfigAdminService](#onos.config.admin.ConfigAdminService)
  

- [Scalar Value Types](#scalar-value-types)



<a name="api/admin/admin.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/admin/admin.proto



<a name="onos.config.admin.Chunk"></a>

### Chunk
Chunk is for streaming a model plugin file to the server.
There is a built in limit in gRPC of 4MB - plugin is usually around 20MB
so break in to chunks of approx 1-2MB.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| so_file | [string](#string) |  | so_file is the name being streamed. |
| content | [bytes](#bytes) |  | content is the bytes content. |






<a name="onos.config.admin.CompactChangesRequest"></a>

### CompactChangesRequest
CompactChangesRequest requests a compaction of the Network Change and Device Change stores


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| retention_period | [google.protobuf.Duration](#google.protobuf.Duration) |  | retention_period is an optional duration of time counting back from the present moment Network changes that were created during this period should not be compacted Any network changes that are older should be compacted If not specified the duration is 0 |






<a name="onos.config.admin.CompactChangesResponse"></a>

### CompactChangesResponse
CompactChangesResponse is a response to the Compact Changes command






<a name="onos.config.admin.GetSnapshotRequest"></a>

### GetSnapshotRequest
GetSnapshotRequest gets the details of a snapshot for a specific device and version.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  | device_id is the ID of a device that has been configured through a NetworkChange. |
| device_version | [string](#string) |  | device version is the semantic version of a device that has been configured through a NetworkChange. |






<a name="onos.config.admin.ListModelsRequest"></a>

### ListModelsRequest
ListModelsRequest carries data for querying registered model plugins.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| verbose | [bool](#bool) |  | verbose option causes all of the ReadWrite and ReadOnly paths to be included. |
| model_name | [string](#string) |  | An optional filter on the name of the model plugins to list. |
| model_version | [string](#string) |  | An optional filter on the version of the model plugins to list |






<a name="onos.config.admin.ListSnapshotsRequest"></a>

### ListSnapshotsRequest
ListSnapshotsRequest requests a list of snapshots for all devices and versions.






<a name="onos.config.admin.ModelInfo"></a>

### ModelInfo
ModelInfo is general information about a model plugin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name given to the model plugin - no spaces and title case. |
| version | [string](#string) |  | version is the semantic version of the Plugin e.g. 1.0.0. |
| model_data | [gnmi.ModelData](#gnmi.ModelData) | repeated | model_data is a set of metadata about the YANG files that went in to generating the model plugin. It includes name, version and organization for each YANG file, similar to how they are represented in gNMI Capabilities. |
| module | [string](#string) |  | module is the name of the Model Plugin on the file system - usually ending in .so.&lt;version&gt;. |
| getStateMode | [uint32](#uint32) |  | getStateMode is flag that defines how the &#34;get state&#34; operation works. 0) means that no retrieval of state is attempted 1) means that the synchronizer will make 2 requests to the device - one for Get with State and another for Get with Operational. 2) means that the synchronizer will do a Get request comprising of each one of the ReadOnlyPaths and their sub paths. If there is a `list` in any one of these paths it will be sent down as is, expecting the devices implementation of gNMI will be able to expand wildcards. 3) means that the synchronizer will do a Get request comprising of each one of the ReadOnlyPaths and their sub paths. If there is a `list` in any one of these paths, a separate call will be made first to find all the instances in the list and a Get including these expanded wildcards will be sent down to the device. |
| read_only_path | [ReadOnlyPath](#onos.config.admin.ReadOnlyPath) | repeated | read_only_path is all of the read only paths for the model plugin. |
| read_write_path | [ReadWritePath](#onos.config.admin.ReadWritePath) | repeated | read_write_path is all of the read write paths for the model plugin. |






<a name="onos.config.admin.ReadOnlyPath"></a>

### ReadOnlyPath
ReadOnlyPath extracted from the model plugin as the definition of a tree of read only items.
In YANG models items are defined as ReadOnly with the `config false` keyword.
This can be applied to single items (leafs) or collections (containers or lists).
When this `config false` is applied to an object every item beneath it will
also become readonly - here these are shown as subpaths.
The complete read only path then will be a concatenation of both e.g.
/cont1a/cont1b-state/list2b/index and the type is defined in the SubPath as UInt8.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | path of the topmost `config false` object e.g. /cont1a/cont1b-state |
| sub_path | [ReadOnlySubPath](#onos.config.admin.ReadOnlySubPath) | repeated | ReadOnlySubPath is a set of children of the path including an entry for the type of the topmost object with subpath `/` An example is /list2b/index |






<a name="onos.config.admin.ReadOnlySubPath"></a>

### ReadOnlySubPath
ReadOnlySubPath is an extension to the ReadOnlyPath to define the datatype of
the subpath


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sub_path | [string](#string) |  | sub_path is the relative path of a child object e.g. /list2b/index |
| value_type | [onos.config.change.device.ValueType](#onos.config.change.device.ValueType) |  | value_type is the datatype of the read only path |






<a name="onos.config.admin.ReadWritePath"></a>

### ReadWritePath
ReadWritePath is extracted from the model plugin as the definition of a writeable attributes.
In YANG models items are writable by default unless they are specified as `config false` or
have an item with `config false` as a parent.
Each configurable item has metadata with meanings taken from the YANG specification RFC 6020.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | path is the full path to the attribute (leaf or leaf-list) |
| value_type | [onos.config.change.device.ValueType](#onos.config.change.device.ValueType) |  | value_type is the data type of the attribute |
| units | [string](#string) |  | units is the unit of measurement e.g. dB, mV |
| description | [string](#string) |  | description is an explaination of the meaning of the attribute |
| mandatory | [bool](#bool) |  | mandatory shows whether the attribute is optional (false) or required (true) |
| default | [string](#string) |  | default is a default value used with optional attributes |
| range | [string](#string) | repeated | range is definition of the range of values a value is allowed |
| length | [string](#string) | repeated | length is a defintion of the length restrictions for the attribute |






<a name="onos.config.admin.RegisterResponse"></a>

### RegisterResponse
RegisterResponse carries status of model plugin registration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is name of the model plugin. |
| version | [string](#string) |  | version is the semantic version of the model plugin. |






<a name="onos.config.admin.RollbackRequest"></a>

### RollbackRequest
RollbackRequest carries the name of a network config to rollback. If there
are subsequent changes to any of the devices in that config, the rollback will
be rejected.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is an optional name of a Network Change to rollback. If no name is given the last network change will be rolled back. If the name given is not of the last network change and error will be given. |
| comment | [string](#string) |  | On optional comment to leave on the rollback. |






<a name="onos.config.admin.RollbackResponse"></a>

### RollbackResponse
RollbackResponse carries the response of the rollback operation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message | [string](#string) |  | A message showing the result of the rollback. |





 


<a name="onos.config.admin.Type"></a>

### Type
Streaming event type

| Name | Number | Description |
| ---- | ------ | ----------- |
| NONE | 0 | NONE indicates this response does not represent a state change |
| ADDED | 1 | ADDED is an event which occurs when an item is added |
| UPDATED | 2 | UPDATED is an event which occurs when an item is updated |
| REMOVED | 3 | REMOVED is an event which occurs when an item is removed |


 

 


<a name="onos.config.admin.ConfigAdminService"></a>

### ConfigAdminService
ConfigAdminService provides means for enhanced interactions with the configuration subsystem.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| UploadRegisterModel | [Chunk](#onos.config.admin.Chunk) stream | [RegisterResponse](#onos.config.admin.RegisterResponse) | UploadRegisterModel uploads and adds the model plugin to the list of supported models. The file is serialized in to Chunks of less than 4MB so as not to break the gRPC byte array limit |
| ListRegisteredModels | [ListModelsRequest](#onos.config.admin.ListModelsRequest) | [ModelInfo](#onos.config.admin.ModelInfo) stream | ListRegisteredModels returns a stream of registered models. |
| RollbackNetworkChange | [RollbackRequest](#onos.config.admin.RollbackRequest) | [RollbackResponse](#onos.config.admin.RollbackResponse) | RollbackNetworkChange rolls back the specified network change (or the latest one). |
| GetSnapshot | [GetSnapshotRequest](#onos.config.admin.GetSnapshotRequest) | [.onos.config.snapshot.device.Snapshot](#onos.config.snapshot.device.Snapshot) | GetSnapshot gets a snapshot for a specific device and version |
| ListSnapshots | [ListSnapshotsRequest](#onos.config.admin.ListSnapshotsRequest) | [.onos.config.snapshot.device.Snapshot](#onos.config.snapshot.device.Snapshot) stream | ListSnapshots gets a list of snapshots across all devices and versions, and streams them back to the caller. The result includes a &#34;replay&#34; of existing snapshots and will watch for any subsequent new changes that come later. |
| CompactChanges | [CompactChangesRequest](#onos.config.admin.CompactChangesRequest) | [CompactChangesResponse](#onos.config.admin.CompactChangesResponse) | CompactChanges requests a snapshot of NetworkChange and DeviceChange stores. This will take all of the Network Changes older than the retention period and flatten them down to just one snapshot (replacing any older snapshot). This will act as a baseline for those changes within the retention period and any future changes. DeviceChanges will be snapshotted to correspond to these NetworkChange compactions leaving an individual snapshot perv device and version combination. |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |


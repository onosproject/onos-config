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
    - [RegisterRequest](#onos.config.admin.RegisterRequest)
    - [RegisterResponse](#onos.config.admin.RegisterResponse)
    - [RollbackRequest](#onos.config.admin.RollbackRequest)
    - [RollbackResponse](#onos.config.admin.RollbackResponse)
    - [SchemaEntry](#onos.config.admin.SchemaEntry)
  
    - [Type](#onos.config.admin.Type)
  
  
    - [ConfigAdminService](#onos.config.admin.ConfigAdminService)
  

- [Scalar Value Types](#scalar-value-types)



<a name="api/admin/admin.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/admin/admin.proto



<a name="onos.config.admin.Chunk"></a>

### Chunk
Chunk is for streaming a model plugin file to the server
There is a built in limit in gRPC of 4MB - plugin is usually around 20MB
so break in to chunks of approx 1MB


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| so_file | [string](#string) |  |  |
| Content | [bytes](#bytes) |  |  |






<a name="onos.config.admin.CompactChangesRequest"></a>

### CompactChangesRequest
CompactChangesRequest requests a compaction of the change stores


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| retention_period | [google.protobuf.Duration](#google.protobuf.Duration) |  |  |






<a name="onos.config.admin.CompactChangesResponse"></a>

### CompactChangesResponse
CompactChangesResponse is a compact response






<a name="onos.config.admin.GetSnapshotRequest"></a>

### GetSnapshotRequest
GetSnapshotRequest gets a snapshot


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| device_version | [string](#string) |  |  |






<a name="onos.config.admin.ListModelsRequest"></a>

### ListModelsRequest
ListModelsRequest carries data for querying registered models.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| verbose | [bool](#bool) |  |  |
| model_name | [string](#string) |  | If blank all are listed |
| model_version | [string](#string) |  | If blank all are listed |






<a name="onos.config.admin.ListSnapshotsRequest"></a>

### ListSnapshotsRequest
ListSnapshotsRequest requests a list of snapshots






<a name="onos.config.admin.ModelInfo"></a>

### ModelInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |
| model_data | [gnmi.ModelData](#gnmi.ModelData) | repeated |  |
| module | [string](#string) |  |  |
| getStateMode | [uint32](#uint32) |  |  |
| read_only_path | [ReadOnlyPath](#onos.config.admin.ReadOnlyPath) | repeated | 6 was used previously |
| read_write_path | [ReadWritePath](#onos.config.admin.ReadWritePath) | repeated |  |






<a name="onos.config.admin.ReadOnlyPath"></a>

### ReadOnlyPath



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| sub_path | [ReadOnlySubPath](#onos.config.admin.ReadOnlySubPath) | repeated |  |






<a name="onos.config.admin.ReadOnlySubPath"></a>

### ReadOnlySubPath



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sub_path | [string](#string) |  |  |
| value_type | [onos.config.change.device.ValueType](#onos.config.change.device.ValueType) |  | from diags.proto |






<a name="onos.config.admin.ReadWritePath"></a>

### ReadWritePath



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| value_type | [onos.config.change.device.ValueType](#onos.config.change.device.ValueType) |  | from diags.proto |
| units | [string](#string) |  |  |
| description | [string](#string) |  |  |
| mandatory | [bool](#bool) |  |  |
| default | [string](#string) |  |  |
| range | [string](#string) | repeated |  |
| length | [string](#string) | repeated |  |






<a name="onos.config.admin.RegisterRequest"></a>

### RegisterRequest
RegisterRequest carries data for registering a YANG model.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| so_file | [string](#string) |  | Full path and filename of a shared object library as a model plugin |






<a name="onos.config.admin.RegisterResponse"></a>

### RegisterResponse
RegisterResponse carries status of YANG model registration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="onos.config.admin.RollbackRequest"></a>

### RollbackRequest
RollbackRequest carries the name of a network config to rollback. If there
are subsequent changes to any of the devices in that config, the rollback will
be rejected. If no name is given the last network change will be rolled back.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| comment | [string](#string) |  |  |






<a name="onos.config.admin.RollbackResponse"></a>

### RollbackResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message | [string](#string) |  |  |






<a name="onos.config.admin.SchemaEntry"></a>

### SchemaEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_path | [string](#string) |  |  |
| schema_json | [string](#string) |  |  |





 


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
| RegisterModel | [RegisterRequest](#onos.config.admin.RegisterRequest) | [RegisterResponse](#onos.config.admin.RegisterResponse) | RegisterModel adds the specified YANG model to the list of supported models. There is no unregister because once loaded plugins cannot be unloaded |
| UploadRegisterModel | [Chunk](#onos.config.admin.Chunk) stream | [RegisterResponse](#onos.config.admin.RegisterResponse) | UploadRegisterModel uploads and adds the model plugin to the list of supported models. |
| ListRegisteredModels | [ListModelsRequest](#onos.config.admin.ListModelsRequest) | [ModelInfo](#onos.config.admin.ModelInfo) stream | ListRegisteredModels returns a stream of registered models. |
| RollbackNetworkChange | [RollbackRequest](#onos.config.admin.RollbackRequest) | [RollbackResponse](#onos.config.admin.RollbackResponse) | RollbackNetworkChange rolls back the specified network change (or the latest one). |
| GetSnapshot | [GetSnapshotRequest](#onos.config.admin.GetSnapshotRequest) | [.onos.config.snapshot.device.Snapshot](#onos.config.snapshot.device.Snapshot) | GetSnapshot gets a snapshot |
| ListSnapshots | [ListSnapshotsRequest](#onos.config.admin.ListSnapshotsRequest) | [.onos.config.snapshot.device.Snapshot](#onos.config.snapshot.device.Snapshot) stream | ListSnapshots gets a list of snapshots |
| CompactChanges | [CompactChangesRequest](#onos.config.admin.CompactChangesRequest) | [CompactChangesResponse](#onos.config.admin.CompactChangesResponse) | CompactChanges requests a snapshot of NetworkChange and DeviceChange stores |

 



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


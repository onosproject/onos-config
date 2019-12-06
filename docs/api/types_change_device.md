# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/change/device/types.proto](#api/types/change/device/types.proto)
    - [Change](#onos.config.change.device.Change)
    - [ChangeValue](#onos.config.change.device.ChangeValue)
    - [DeviceChange](#onos.config.change.device.DeviceChange)
    - [NetworkChangeRef](#onos.config.change.device.NetworkChangeRef)
    - [PathValue](#onos.config.change.device.PathValue)
    - [TypedValue](#onos.config.change.device.TypedValue)
  
    - [ValueType](#onos.config.change.device.ValueType)
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="api/types/change/device/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/change/device/types.proto



<a name="onos.config.change.device.Change"></a>

### Change
Change represents a configuration change to a single device


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  | &#39;device_id&#39; is the identifier of the device to which this change applies |
| device_version | [string](#string) |  | &#39;device_version&#39; is an optional device version to which to apply this change |
| device_type | [string](#string) |  | &#39;device_type&#39; is an optional device type to which to apply this change |
| values | [ChangeValue](#onos.config.change.device.ChangeValue) | repeated | &#39;values&#39; is a set of change values to apply |






<a name="onos.config.change.device.ChangeValue"></a>

### ChangeValue
ChangeValue is an individual Path/Value and removed flag combination in a Change


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | &#39;path&#39; is the path to change |
| value | [TypedValue](#onos.config.change.device.TypedValue) |  | &#39;value&#39; is the change value |
| removed | [bool](#bool) |  | &#39;removed&#39; indicates whether this is a delete |






<a name="onos.config.change.device.DeviceChange"></a>

### DeviceChange
DeviceChange is a stored configuration change for a single device


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the unique identifier of the change |
| index | [uint64](#uint64) |  | &#39;index&#39; is a monotonically increasing, globally unique index of the change The index is provided by the store, is static and unique for each unique change identifier, and should not be modified by client code. |
| revision | [uint64](#uint64) |  | &#39;revision&#39; is the change revision number The revision number is provided by the store and should not be modified by client code. Each unique state of the change will be assigned a unique revision number which can be used for optimistic concurrency control when updating or deleting the change state. |
| attempt | [uint32](#uint32) |  | &#39;attempt&#39; indicates the number of attempts to apply the change |
| network_change | [NetworkChangeRef](#onos.config.change.device.NetworkChangeRef) |  | &#39;network_change&#39; is a reference to the NetworkChange that created this change |
| change | [Change](#onos.config.change.device.Change) |  | &#39;change&#39; is the change object |
| status | [onos.config.change.Status](#onos.config.change.Status) |  | &#39;status&#39; is the lifecycle status of the change |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;created&#39; is the time at which the change was created |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;updated&#39; is the time at which the change was last updated |






<a name="onos.config.change.device.NetworkChangeRef"></a>

### NetworkChangeRef
NetworkChangeRef is a back reference to the NetworkChange that created a DeviceChange


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the identifier of the network change from which this change was created |
| index | [uint64](#uint64) |  | &#39;index&#39; is the index of the network change from which this change was created |






<a name="onos.config.change.device.PathValue"></a>

### PathValue
PathValue is an individual Path/Value combination - it is like ChangeValue above
without the removed flag - it is not used in the DeviceChange store
Instead it is useful for handling OpState and Snapshots where `removed` is not relevant


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | &#39;path&#39; is the path to change |
| value | [TypedValue](#onos.config.change.device.TypedValue) |  | &#39;value&#39; is the change value |






<a name="onos.config.change.device.TypedValue"></a>

### TypedValue
TypedValue is a value represented as a byte array


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bytes | [bytes](#bytes) |  | &#39;bytes&#39; is the bytes array |
| type | [ValueType](#onos.config.change.device.ValueType) |  | &#39;type&#39; is the value type |
| type_opts | [int32](#int32) | repeated | &#39;type_opts&#39; is a set of type options |





 


<a name="onos.config.change.device.ValueType"></a>

### ValueType
ValueType is the type for a value

| Name | Number | Description |
| ---- | ------ | ----------- |
| EMPTY | 0 |  |
| STRING | 1 |  |
| INT | 2 |  |
| UINT | 3 |  |
| BOOL | 4 |  |
| DECIMAL | 5 |  |
| FLOAT | 6 |  |
| BYTES | 7 |  |
| LEAFLIST_STRING | 8 |  |
| LEAFLIST_INT | 9 |  |
| LEAFLIST_UINT | 10 |  |
| LEAFLIST_BOOL | 11 |  |
| LEAFLIST_DECIMAL | 12 |  |
| LEAFLIST_FLOAT | 13 |  |
| LEAFLIST_BYTES | 14 |  |


 

 

 



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


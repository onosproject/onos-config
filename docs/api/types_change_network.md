# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/change/network/types.proto](#api/types/change/network/types.proto)
    - [DeviceChangeRef](#onos.config.change.network.DeviceChangeRef)
    - [NetworkChange](#onos.config.change.network.NetworkChange)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api/types/change/network/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/change/network/types.proto



<a name="onos.config.change.network.DeviceChangeRef"></a>

### DeviceChangeRef
DeviceChangeRef is a reference to a device change


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_change_id | [string](#string) |  | &#39;device_change_id&#39; is the unique identifier of the device change |






<a name="onos.config.change.network.NetworkChange"></a>

### NetworkChange
NetworkChange specifies the configuration for a network change
A network change is a configuration change that spans multiple devices. The change contains a list of
per-device changes to be applied to the network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the unique identifier of the change This field should be set prior to persisting the object. |
| index | [uint64](#uint64) |  | &#39;index&#39; is a monotonically increasing, globally unique index of the change The index is provided by the store, is static and unique for each unique change identifier, and should not be modified by client code. |
| revision | [uint64](#uint64) |  | &#39;revision&#39; is the change revision number The revision number is provided by the store and should not be modified by client code. Each unique state of the change will be assigned a unique revision number which can be used for optimistic concurrency control when updating or deleting the change state. |
| status | [onos.config.change.Status](#onos.config.change.Status) |  | &#39;status&#39; is the current lifecycle status of the change |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;created&#39; is the time at which the change was created |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;updated&#39; is the time at which the change was last updated |
| changes | [onos.config.change.device.Change](#onos.config.change.device.Change) | repeated | &#39;changes&#39; is a set of changes to apply to devices The list of changes should contain only a single change per device/version pair. |
| refs | [DeviceChangeRef](#onos.config.change.network.DeviceChangeRef) | repeated | &#39;refs&#39; is a set of references to stored device changes |
| deleted | [bool](#bool) |  | &#39;deleted&#39; is a flag indicating whether this change is being deleted by a snapshot |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |


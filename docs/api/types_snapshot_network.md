# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/snapshot/network/types.proto](#api/types/snapshot/network/types.proto)
    - [DeviceSnapshotRef](#onos.config.snapshot.network.DeviceSnapshotRef)
    - [NetworkSnapshot](#onos.config.snapshot.network.NetworkSnapshot)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api/types/snapshot/network/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/snapshot/network/types.proto



<a name="onos.config.snapshot.network.DeviceSnapshotRef"></a>

### DeviceSnapshotRef
DeviceSnapshotRef is a reference to a device snapshot


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_snapshot_id | [string](#string) |  | &#39;device_snapshot_id&#39; is the unique identifier of the device snapshot |






<a name="onos.config.snapshot.network.NetworkSnapshot"></a>

### NetworkSnapshot
NetworkSnapshot is a snapshot of all network changes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the unique snapshot identifier |
| index | [uint64](#uint64) |  | &#39;index&#39; is a monotonically increasing, globally unique snapshot index |
| revision | [uint64](#uint64) |  | &#39;revision&#39; is the request revision number |
| status | [onos.config.snapshot.Status](#onos.config.snapshot.Status) |  | &#39;status&#39; is the snapshot status |
| retention | [onos.config.snapshot.RetentionOptions](#onos.config.snapshot.RetentionOptions) |  | &#39;retention&#39; specifies the duration for which to retain changes |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;created&#39; is the time at which the configuration was created |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;updated&#39; is the time at which the configuration was last updated |
| refs | [DeviceSnapshotRef](#onos.config.snapshot.network.DeviceSnapshotRef) | repeated | &#39;refs&#39; is a set of references to stored device snapshots |





 

 

 

 



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


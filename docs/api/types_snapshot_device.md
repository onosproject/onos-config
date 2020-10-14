# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/snapshot/device/types.proto](#api/types/snapshot/device/types.proto)
    - [DeviceSnapshot](#onos.config.snapshot.device.DeviceSnapshot)
    - [NetworkSnapshotRef](#onos.config.snapshot.device.NetworkSnapshotRef)
    - [Snapshot](#onos.config.snapshot.device.Snapshot)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api/types/snapshot/device/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/snapshot/device/types.proto



<a name="onos.config.snapshot.device.DeviceSnapshot"></a>

### DeviceSnapshot
DeviceSnapshot is a device snapshot


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the unique snapshot identifier |
| device_id | [string](#string) |  | &#39;device_id&#39; is the device to which the snapshot applies |
| device_version | [string](#string) |  | &#39;device_version&#39; is the version to which the snapshot applies |
| device_type | [string](#string) |  | &#39;device_type&#39; is an optional device type to which to apply this change |
| revision | [uint64](#uint64) |  | &#39;revision&#39; is the request revision number |
| network_snapshot | [NetworkSnapshotRef](#onos.config.snapshot.device.NetworkSnapshotRef) |  | &#39;network_snapshot&#39; is a reference to the network snapshot from which this snapshot was created |
| max_network_change_index | [uint64](#uint64) |  | &#39;max_network_change_index&#39; is the maximum network change index to be snapshotted for the device |
| status | [onos.config.snapshot.Status](#onos.config.snapshot.Status) |  | &#39;status&#39; is the snapshot status |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;created&#39; is the time at which the configuration was created |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | &#39;updated&#39; is the time at which the configuration was last updated |






<a name="onos.config.snapshot.device.NetworkSnapshotRef"></a>

### NetworkSnapshotRef
NetworkSnapshotRef is a back reference to the NetworkSnapshot that created a DeviceSnapshot


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is the identifier of the network snapshot from which this snapshot was created |
| index | [uint64](#uint64) |  | &#39;index&#39; is the index of the network snapshot from which this snapshot was created |






<a name="onos.config.snapshot.device.Snapshot"></a>

### Snapshot
Snapshot is a snapshot of the state of a single device


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | &#39;id&#39; is a unique snapshot identifier |
| device_id | [string](#string) |  | &#39;device_id&#39; is the device to which the snapshot applies |
| device_version | [string](#string) |  | &#39;device_version&#39; is the version to which the snapshot applies |
| device_type | [string](#string) |  | &#39;device_type&#39; is an optional device type to which to apply this change |
| snapshot_id | [string](#string) |  | &#39;snapshot_id&#39; is the ID of the snapshot |
| change_index | [uint64](#uint64) |  | &#39;change_index&#39; is the change index at which the snapshot ended |
| values | [onos.config.change.device.PathValue](#onos.config.change.device.PathValue) | repeated | &#39;values&#39; is a list of values to set |





 

 

 

 



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


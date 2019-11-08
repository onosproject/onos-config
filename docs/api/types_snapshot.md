# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/snapshot/types.proto](#api/types/snapshot/types.proto)
    - [RetentionOptions](#onos.config.snapshot.RetentionOptions)
    - [Status](#onos.config.snapshot.Status)
  
    - [Phase](#onos.config.snapshot.Phase)
    - [State](#onos.config.snapshot.State)
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="api/types/snapshot/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/snapshot/types.proto



<a name="onos.config.snapshot.RetentionOptions"></a>

### RetentionOptions
RetentionOptions specifies the retention policy for a change log


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| retain_window | [google.protobuf.Duration](#google.protobuf.Duration) |  | &#39;retain_window&#39; is the duration for which to retain network changes |






<a name="onos.config.snapshot.Status"></a>

### Status
Status is the status of a snapshot


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [Phase](#onos.config.snapshot.Phase) |  | &#39;phase&#39; is the snapshot phase |
| state | [State](#onos.config.snapshot.State) |  | &#39;state&#39; is the state of a snapshot |





 


<a name="onos.config.snapshot.Phase"></a>

### Phase
Phase is a snapshot phase

| Name | Number | Description |
| ---- | ------ | ----------- |
| MARK | 0 | MARK is the first phase in which changes are marked for deletion |
| DELETE | 1 | DELETE is the second phase in which changes are deleted from stores |



<a name="onos.config.snapshot.State"></a>

### State
State is the state of a snapshot within a phase

| Name | Number | Description |
| ---- | ------ | ----------- |
| PENDING | 0 | PENDING indicates the snapshot is pending |
| RUNNING | 1 | RUNNING indicates the snapshot is in progress |
| COMPLETE | 2 | COMPLETE indicates the snapshot is complete |


 

 

 



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


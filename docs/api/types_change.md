# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/types/change/types.proto](#api/types/change/types.proto)
    - [Status](#onos.config.change.Status)
  
    - [Phase](#onos.config.change.Phase)
    - [Reason](#onos.config.change.Reason)
    - [State](#onos.config.change.State)
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="api/types/change/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/types/change/types.proto



<a name="onos.config.change.Status"></a>

### Status
Status is the status of a NetworkChange


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [Phase](#onos.config.change.Phase) |  | &#39;phase&#39; is the current phase of the NetworkChange |
| state | [State](#onos.config.change.State) |  | &#39;state&#39; is the state of the change within a Phase |
| reason | [Reason](#onos.config.change.Reason) |  | &#39;reason&#39; is a failure reason |
| message | [string](#string) |  | message is a result message |
| incarnation | [uint64](#uint64) |  | incarnation is the status incarnation number |





 


<a name="onos.config.change.Phase"></a>

### Phase
Phase is the phase of a NetworkChange

| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE | 0 | CHANGE indicates the change has been requested |
| ROLLBACK | 1 | ROLLBACK indicates a rollback has been requested for the change |



<a name="onos.config.change.Reason"></a>

### Reason
Reason is a reason for a FAILED state

| Name | Number | Description |
| ---- | ------ | ----------- |
| NONE | 0 | NONE indicates no error has occurred |
| ERROR | 1 | ERROR indicates an error occurred when applying the change |



<a name="onos.config.change.State"></a>

### State
State is the state of a phase

| Name | Number | Description |
| ---- | ------ | ----------- |
| PENDING | 0 | PENDING indicates the phase is pending |
| COMPLETE | 2 | COMPLETE indicates the phase is complete |
| FAILED | 3 | FAILED indicates the phase failed |


 

 

 



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


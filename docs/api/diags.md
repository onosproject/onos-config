# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/diags/diags.proto](#api/diags/diags.proto)
    - [ListDeviceChangeRequest](#onos.config.diags.ListDeviceChangeRequest)
    - [ListDeviceChangeResponse](#onos.config.diags.ListDeviceChangeResponse)
    - [ListNetworkChangeRequest](#onos.config.diags.ListNetworkChangeRequest)
    - [ListNetworkChangeResponse](#onos.config.diags.ListNetworkChangeResponse)
    - [OpStateRequest](#onos.config.diags.OpStateRequest)
    - [OpStateResponse](#onos.config.diags.OpStateResponse)
  
    - [Type](#onos.config.diags.Type)
  
    - [ChangeService](#onos.config.diags.ChangeService)
    - [OpStateDiags](#onos.config.diags.OpStateDiags)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api/diags/diags.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/diags/diags.proto



<a name="onos.config.diags.ListDeviceChangeRequest"></a>

### ListDeviceChangeRequest
ListDeviceChangeRequest requests a stream of changes and updates to them
By default, the request requests a stream of all changes that are present in the topology when
the request is received by the service. However, if `subscribe` is `true`, the stream will remain
open after all changes have been sent and events that occur following the last changes will be
streamed to the client until the stream is closed.
If &#34;withoutReplay&#34; is true then only changes that happen after the call will be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscribe | [bool](#bool) |  | subscribe indicates whether to subscribe to events (e.g. ADD, UPDATE, and REMOVE) that occur after all devices have been streamed to the client |
| device_id | [string](#string) |  | option to specify a specific device change - if blank or &#39;*&#39; then select all Can support `*` (match many chars) or &#39;?&#39; (match one char) as wildcard |
| device_version | [string](#string) |  | device_version is an optional device version |
| withoutReplay | [bool](#bool) |  | option to request only changes that happen after the call |






<a name="onos.config.diags.ListDeviceChangeResponse"></a>

### ListDeviceChangeResponse
ListDeviceChangeResponse carries a single network change event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| change | [onos.config.change.device.DeviceChange](#onos.config.change.device.DeviceChange) |  | change is the device change on which the event occurred |
| type | [Type](#onos.config.diags.Type) |  | type is a qualification of the type of change being made |






<a name="onos.config.diags.ListNetworkChangeRequest"></a>

### ListNetworkChangeRequest
ListNetworkChangeRequest requests a stream of changes and updates to them
By default, the request requests a stream of all changes that are present in the topology when
the request is received by the service. However, if `subscribe` is `true`, the stream will remain
open after all changes have been sent and events that occur following the last changes will be
streamed to the client until the stream is closed.
If &#34;withoutReplay&#34; is true then only changes that happen after the call will be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscribe | [bool](#bool) |  | subscribe indicates whether to subscribe to events (e.g. ADD, UPDATE, and REMOVE) that occur after all devices have been streamed to the client |
| changeid | [string](#string) |  | option to specify a specific network change - if blank or &#39;*&#39; then select all Can support `*` (match many chars) or &#39;?&#39; (match one char) as wildcard |
| withoutReplay | [bool](#bool) |  | option to request only changes that happen after the call |






<a name="onos.config.diags.ListNetworkChangeResponse"></a>

### ListNetworkChangeResponse
ListNetworkChangeResponse carries a single network change event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| change | [onos.config.change.network.NetworkChange](#onos.config.change.network.NetworkChange) |  | change is the network change on which the event occurred |
| type | [Type](#onos.config.diags.Type) |  | type is a qualification of the type of change being made |






<a name="onos.config.diags.OpStateRequest"></a>

### OpStateRequest
OpStateRequest is a message for specifying GetOpState query parameters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deviceId | [string](#string) |  | The request is always in the context of a Device ID. If the device does not exist or is disconnected an error will be returned. |
| subscribe | [bool](#bool) |  | subscribe indicates whether to subscribe to events (e.g. ADD, UPDATE, and REMOVE) that occur after all paths for the device have been streamed to the client |






<a name="onos.config.diags.OpStateResponse"></a>

### OpStateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [onos.config.admin.Type](#onos.config.admin.Type) |  | type is the type of the event |
| pathvalue | [onos.config.change.device.PathValue](#onos.config.change.device.PathValue) |  | device is the device on which the event occurred |





 


<a name="onos.config.diags.Type"></a>

### Type
Change (Network or Device) event type

| Name | Number | Description |
| ---- | ------ | ----------- |
| NONE | 0 | NONE indicates this response does not represent a modification of the Change |
| ADDED | 1 | ADDED is an event which occurs when a Change is added to the topology |
| UPDATED | 2 | UPDATED is an event which occurs when a Change is updated |
| REMOVED | 3 | REMOVED is an event which occurs when a Change is removed from the configuration |


 

 


<a name="onos.config.diags.ChangeService"></a>

### ChangeService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListNetworkChanges | [ListNetworkChangeRequest](#onos.config.diags.ListNetworkChangeRequest) | [ListNetworkChangeResponse](#onos.config.diags.ListNetworkChangeResponse) stream | List gets a stream of network change add/update/remove events for network changes matching changeid |
| ListDeviceChanges | [ListDeviceChangeRequest](#onos.config.diags.ListDeviceChangeRequest) | [ListDeviceChangeResponse](#onos.config.diags.ListDeviceChangeResponse) stream | List gets a stream of device change add/update/remove events for device changes matching changeid |


<a name="onos.config.diags.OpStateDiags"></a>

### OpStateDiags
OpStateDiags provides means for obtaining diagnostic information about internal system state.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetOpState | [OpStateRequest](#onos.config.diags.OpStateRequest) | [OpStateResponse](#onos.config.diags.OpStateResponse) stream | GetOpState returns a stream of submitted OperationalStateCache aimed at individual devices. If subscribe is true keep on streaming after the initial set are finished |

 



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


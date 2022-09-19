# EtherNet-IP Go Adapter
The EtherNet-IP adapter functions as a EtherNet-IP Client and allows an IoT Gateway (or any other client) to interact with an EtherNet-IP server.

Communication with the EtherNet-IP Adapter is enabled through MQTT and configuration Collections which are detailed below.

Currently, Read and Write are supported by the adapter, with more options to follow.

## ClearBlade Platform Dependencies
The EtherNet-IP adapter adapter was constructed to provide the ability to communicate with a _System_ defined in a ClearBlade Platform instance. Therefore, the adapter requires a _System_ to have been created within a ClearBlade Platform instance.

Once a System has been created, artifacts must be defined within the ClearBlade Platform system to allow the adapter to function properly. At a minimum: 

  * An adapter configuration data collection named `adapter_config` needs to be created in the ClearBlade Platform _system_ and populated with data appropriate to the modbus client adapter adapter. The schema of the data collection should be as follows:

| Column Name      | Column Datatype |
| ---------------- | --------------- |
| adapter_name     | string          |
| adapter_settings | string          |
| topic_root       | string          |

## Adapter Settings Structure
The `adapter_settings` JSON string provided in the `adapter_config` collection is expected to have the following structure. This JSON is how you provide the Adapter with the specific server connection details.

```json
{
  "endpoint_ip": "10.10.10.10",
  "endpoint_tcp_port": 44818,
}
```

### Supported Operations
| Operation |
| ---------------- |
| `read` |
| `write` | 

## MQTT Topic Structure
The OPC UA adapter will subscribe to a specific topics in order to handle OPC UA operations. Additionally, the adapter will publish messages to MQTT topics for results of the OPC UA operations. The topic structures utilized are as follows:

 * OPC UA Read Request: {__TOPIC ROOT__}/read
 * OPC UA Read Results: {__TOPIC ROOT__}/read/response
 * OPC UA Write Request: {__TOPIC ROOT__}/write
 * OPC UA Write Response: {__TOPIC ROOT__}/write/response
 * OPC UA Method Request: {__TOPIC ROOT__}/method
 * OPC UA Method Response: {__TOPIC ROOT__}/method/response
 * OPC UA Subscribe Request: {__TOPIC ROOT__}/subscribe
   ** create, publish, and delete are the only supported services in the opcua library being utilized
 * OPC UA Subscribe Response: {__TOPIC ROOT__}/subscribe/response
 * OPC UA Publish: {__TOPIC ROOT__}/publish/response
## MQTT Message Structure

### EtherNet-IP Read Request Payload Format
```json
{
  "tags": ["tag1", "tag2", "tag3"]
}
```

### EtherNet-IP Read Results Payload Format
 ```json
 {
  "server_timestamp": "2021-07-30T05:04:55Z",
  "data": {
    "tag1": {
      "value": 6,
      "source_timestamp": "2021-07-30T05:04:55Z"
    },
    "tag2": {
      "value": -1.698198,
      "source_timestamp": "2021-07-30T05:04:55Z"
    },
    "tag3": {
      "value": -2,
      "source_timestamp": "2021-07-30T05:04:55Z"
    }
  },
  "success": true,
  "status_code": 0,
  "error_message": ""
}
 ```

### EtherNet-IP Write Request Payload Format
```json
{
    "node_id": "ns=3;i=1001",
    "value": 25 // can be string or int/float
}
```

### EtherNet-IP Write Response Payload Format
```json
{
    "node_id": "ns=3;i=1001",
    "timestamp": "", //ISO formatted timestamp
    "success": true|false,
    "status_code": 0, //Integer
    "error_message": ""
}
```

## Starting the adapter
This adapter is built using the [adapter-go-library](https://github.com/ClearBlade/adapter-go-library) which allows for multiple options for starting the adapter, including CLI flags and environment variables. It is recommended to use a device service account for authentication with this adapter. See the below chart for available start options as well as their defaults.

All ClearBlade Adapters require a certain set of System specific variables to start and connect with the ClearBlade Platform/Edge. This library allows these to be passed in either by command line arguments, or environment variables. Note that command line arguments take precedence over environment variables.

| Name | CLI Flag | Environment Variable | Default |
| --- | --- | --- | --- |
| System Key | `systemKey` | `CB_SYSTEM_KEY` | N/A |
| System Secret | `systemSecret` | `CB_SYSTEM_SECRET` | N/A |
| Platform/Edge URL | `platformURL` | N/A | `http://localhost:9000` |
| Platform/Edge Messaging URL | `messagingURL` | N/A | `localhost:1883` |
| Device Name (**depreciated**) | `deviceName` | N/A | `adapterName` provided when calling `adapter_library.ParseArguments` |
| Device Password/Active Key (**depreciated)** | `password` | N/A | N/A |
| Device Service Account | N/A | `CB_SERVICE_ACCOUNT` | N/A |
| Device Service Account Token | N/A | `CB_SERVICE_ACCOUNT_TOKEN` | N/A |
| Log Level | `logLevel` | N/A | `info` |
| Adapter Config Collection Name | `adapterConfigCollection` | N/A | `adapter_config` |


`ethernet-ip-go-adapter -systemKey=<SYSTEM_KEY> -systemSecret=<SYSTEM_SECRET> -platformURL=<PLATFORM_URL> -messagingURL=<MESSAGING_URL> -deviceName=<DEVICE_NAME> -password=<DEVICE_ACTIVE_KEY> -adapterConfigCollection=<COLLECTION_NAME> -logLevel=<LOG_LEVEL>`


A System Key and System Secret will always be required to start the adapter, and it's recommended to always use a Device Service Account & Token for Adapters. 

Device Name and Password for Adapters are **depreciated** and only provided for backwards compatibility and should not be used for any new adapters.

## Setup
---
The EtherNet-IP Go adapter is dependent upon the ClearBlade Go SDK and its dependent libraries being installed. The OPC UA Go adapter was written in Go and therefore requires Go to be installed (https://golang.org/doc/install).

### Adapter compilation
In order to compile the adapter for execution, the following steps need to be performed:

 1. Retrieve the adapter source code  
    * ```git clone git@github.com:ClearBlade/ethernet-ip-go-adapter.git```
 2. Navigate to the _opc-ua-go-adapter_ directory  
    * ```cd ethernet-ip-go-adapter```
 3. Compile the adapter for your needed architecture and OS
    * ```GOARCH=arm GOARM=5 GOOS=linux go build```


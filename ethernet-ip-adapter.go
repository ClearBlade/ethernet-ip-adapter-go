package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	adapter_library "github.com/clearblade/adapter-go-library"
	mqttTypes "github.com/clearblade/mqtt_parsing"
	eip "github.com/loki-os/go-ethernet-ip"
)

const (
	adapterName    = "ethernet-ip-adapter"
	appuri         = "urn:cb-opc-ua-adapter:client"
	readTopic      = "read"
	writeTopic     = "write"
	methodTopic    = "method"
	subscribeTopic = "subscribe"
	publishTopic   = "publish"
)

var (
	adapterSettings *ethernetIpAdapterSettings
	adapterConfig   *adapter_library.AdapterConfig
	eipClient       *eip.EIPTCP
	eipConfig       *eip.Config
	eipTagMap       map[string]*eip.Tag
)

func main() {
	err := adapter_library.ParseArguments(adapterName)
	if err != nil {
		log.Fatalf("[FATAL] Failed to parse arguments: %s\n", err.Error())
	}

	adapterConfig, err = adapter_library.Initialize()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize: %s\n", err.Error())
	}

	adapterSettings = &ethernetIpAdapterSettings{}
	err = json.Unmarshal([]byte(adapterConfig.AdapterSettings), adapterSettings)
	if err != nil {
		log.Fatalf("[FATAL] Failed to parse Adapter Settings %s\n", err.Error())
	}

	err = adapter_library.ConnectMQTT(adapterConfig.TopicRoot+"/#", cbMessageHandler)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect MQTT: %s\n", err.Error())
	}

	// initialize ethernet IP connection
	initializeEIP()

	//TODO - Add an interval to refresh the tags

	// wait for signal to stop/kill process to allow for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c

	log.Printf("[INFO] OS signal %s received, gracefully shutting down adapter.\n", sig)
	os.Exit(0)

}

func initializeEIP() {
	var err error

	//Create the default config
	eipConfig = eip.DefaultConfig()

	//Create TCP Connection
	log.Printf("[INFO] Creating connection to EtherNet-IP server address %s:%d\n", adapterSettings.EndpointIp, adapterSettings.EndpointPort)

	eipConfig.TCPPort = uint16(adapterSettings.EndpointPort)

	eipClient, err = eip.NewTCP(adapterSettings.EndpointIp, eipConfig)
	if err != nil {
		// cannot resolve host
		log.Fatalln(err)
	}

	//Connect to server using TCP
	log.Printf("[INFO] Connecting to EtherNet-IP server\n")
	err = eipClient.Connect()
	if err != nil {
		// cannot connect to host
		log.Fatalln(err)
	}

	//Retrieve all tags and populate tag map
	log.Printf("[INFO] Retrieving device tags\n")
	eipTagMap, err = eipClient.AllTags()
	if err != nil {
		// cannot get tags
		log.Fatalln(err)
	}
	log.Printf("[DEBUG] Tags retrieved: %#v\n", eipTagMap)
}

func cbMessageHandler(message *mqttTypes.Publish) {
	//Determine the type of request that was received
	if strings.Contains(message.Topic.Whole, "response") {
		log.Println("[DEBUG] cbMessageHandler - Received response, ignoring")
	} else if strings.Contains(message.Topic.Whole, readTopic) {
		log.Println("[INFO] cbMessageHandler - Received Ethernet-IP read request")
		go handleReadRequest(message)
	} else if strings.Contains(message.Topic.Whole, writeTopic) {
		log.Println("[INFO] cbMessageHandler - Received Ethernet-IP write request")
		go handleWriteRequest(message)
	} else {
		log.Printf("[ERROR] cbMessageHandler - Unknown request received: topic = %s, payload = %#v\n", message.Topic.Whole, message.Payload)
	}
}

// OPC UA Attribute Service Set - read
func handleReadRequest(message *mqttTypes.Publish) {

	mqttResp := ethernetIpReadResponseMQTTMessage{
		ServerTimestamp: "",
		Data:            make(map[string]ethernetIpReadResponseData),
		Success:         true,
		StatusCode:      0,
		ErrorMessage:    "",
	}

	readReq := ethernetIpReadRequestMQTTMessage{}
	err := json.Unmarshal(message.Payload, &readReq)
	if err != nil {
		log.Printf("[ERROR] Failed to unmarshal request JSON: %s\n", err.Error())
		returnReadError(err.Error(), &mqttResp)
		return
	}

	mqttResp.ServerTimestamp = time.Now().Format(time.RFC3339)

	for _, tag := range readReq.Tags {
		if _, ok := eipTagMap[tag]; ok {
			mqttResp.Data[tag], err = readTag(eipTagMap[tag])
			if err != nil {
				log.Printf("[ERROR] Error reading tag: %s\n", err.Error())
				returnReadError(err.Error(), &mqttResp)
			}
		} else {
			log.Printf("[ERROR] Cannot read tag, tag does not exist %s", tag)
			returnReadError(err.Error(), &mqttResp)
			return
		}
	}

	// 	opcuaResp, err := opcuaClient.Read(opcuaReadReq)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Read request failed: %s\n", err.Error())
	// 		returnReadError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	for idx, result := range opcuaResp.Results {
	// 		if result.Status == ua.StatusOK {
	// 			mqttResp.ServerTimestamp = result.ServerTimestamp.Format(time.RFC3339)
	// 			mqttResp.Data[readReq.NodeIDs[idx]] = opcuaReadResponseData{
	// 				Value:           result.Value.Value(),
	// 				SourceTimestamp: result.SourceTimestamp.Format(time.RFC3339),
	// 			}
	// 		} else {
	// 			log.Printf("[ERROR] Read Status not OK for node id %s: %+v\n", readReq.NodeIDs[idx], result.Status)
	// 			returnReadError(fmt.Sprintf("Read Status not OK for node id %s: %+v\n", readReq.NodeIDs[idx], result.Status), &mqttResp)
	// 			return
	// 		}
	// 	}

	// 	if len(mqttResp.Data) == 0 {
	// 		log.Println("[IFNO] No data received, nothing to publish")
	// 		return
	// 	}

	// 	publishJson(adapterConfig.TopicRoot+"/"+readTopic+"/response", mqttResp)
}

func readTag(tag *eip.Tag) (ethernetIpReadResponseData, error) {
	readResp := ethernetIpReadResponseData{}

	err := tag.Read()
	if err != nil {
		// cannot read tag
		return readResp, err
	}

	// From https://www.odva.org/wp-content/uploads/2020/06/PUB00123R1_Common-Industrial_Protocol_and_Family_of_CIP_Networks.pdf
	//
	// 2.9.2. Data Types
	// Data types (first byte = 0xA0–0xDF) can be either structured (first byte = 0xA0–0xA3, 0xA8 or 0xB0) or elementary (first and only byte = 0xC1–0xDE). All other values are reserved. structured data types can be arrays of
	// elementary data types or a collection of arrays or elementary data types. Of particular importance in the context
	// of this book are elementary data types, which are used within EDS files to specify the data types of parameters
	// and other entities.
	// Here is a list of commonly used data types:
	// - 1-bit (encoded into 1 byte):
	//  • Boolean, BOOL, Type Code 0xC1;
	// - 1-byte:
	//  • Bit string, 8 bits, BYTE, Type Code 0xD1;
	//  • Unsigned 8-bit integer, USINT, Type Code 0xC6;
	//  • Signed 8-bit integer, SINT, Type Code 0xC2;
	// - 2-byte:
	//  • Bit string, 16-bits, WORD, Type Code 0xD2;
	//  • Unsigned 16-bit integer, UINT, Type Code 0xC7;
	//  • Signed 16-bit integer, INT, Type Code 0xC3;
	// - 4-byte:
	//  • Bit string, 32 bits, DWORD, Type Code 0xD3;
	//  • Unsigned 32-bit integer, UDINT, Type Code 0xC8;
	//  • Signed 32-bit integer, DINT, Type Code 0xC4.

	switch tag.Type {
	case eip.NULL:
		readResp.Value = nil
	case eip.BOOL:
		readResp.Value, _ = strconv.ParseBool(string(tag.Int32()))
	case eip.SINT:
	case eip.INT:
	case eip.DINT:
	case eip.USINT: //unsigned small int (1 byte)
	case eip.UINT: //unsigned int (2 bytes)
	case eip.UDINT: //unsigned double integer (4 bytes)
		readResp.Value = tag.Int32()
	//case eip.LINT:
	// case eip.ULINT: //unsigned long int (8 bytes)
	// case eip.REAL: //Real number (4 bytes)
	// case eip.LREAL:
	case eip.STRING:
		readResp.Value = tag.String()
	default:
		return readResp, fmt.Errorf("unsupported data type: %d", tag.Type)
	}

	readResp.SourceTimestamp = time.Now().UTC().Format(time.RFC3339) //time.Now().Format(JavascriptISOString)
	return readResp, nil
}

//OPC UA Attribute Service Set - write
func handleWriteRequest(message *mqttTypes.Publish) {

	// 	mqttResp := opcuaWriteResponseMQTTMessage{
	// 		NodeID:       "",
	// 		Timestamp:    "",
	// 		Success:      true,
	// 		StatusCode:   0,
	// 		ErrorMessage: "",
	// 	}

	// 	writeReq := opcuaWriteRequestMQTTMessage{}
	// 	err := json.Unmarshal(message.Payload, &writeReq)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Failed to unmarshal request JSON: %s\n", err.Error())
	// 		returnWriteError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	id, err := ua.ParseNodeID(writeReq.NodeID)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Failed to parse OPC UA Node ID: %s\n", err.Error())
	// 		returnWriteError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	nodeType, err := getTagDataType(id)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Failed to get type for Node ID %s: %s\n", id.String(), err.Error())
	// 		returnWriteError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	switch val := writeReq.Value.(type) {
	// 	case []interface{}:
	// 		switch *nodeType {
	// 		case ua.TypeIDBoolean:
	// 			convertedArray := make([]bool, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(bool))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDDouble:
	// 			convertedArray := make([]float64, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(float64))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDInt16:
	// 			convertedArray := make([]int16, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(int16))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDInt32:
	// 			convertedArray := make([]int32, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(int32))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDInt64:
	// 			convertedArray := make([]int64, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(int64))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDString:
	// 			convertedArray := make([]string, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(string))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		case ua.TypeIDFloat:
	// 			convertedArray := make([]float32, 0)
	// 			for _, i := range val {
	// 				v, err := getConvertedValue(nodeType, i)
	// 				if err != nil {
	// 					log.Println("[ERROR] " + err.Error())
	// 					returnWriteError(err.Error(), &mqttResp)
	// 					return
	// 				}
	// 				convertedArray = append(convertedArray, v.(float32))
	// 			}
	// 			writeReq.Value = convertedArray
	// 		default:
	// 			log.Printf("[ERROR] Unhandled node type: " + nodeType.String())
	// 			return
	// 		}
	// 	case interface{}:
	// 		_, err = getConvertedValue(nodeType, val)
	// 		if err != nil {
	// 			log.Println("[ERROR] " + err.Error())
	// 			returnWriteError(err.Error(), &mqttResp)
	// 			return
	// 		}
	// 	default:
	// 		log.Printf("[ERROR] Unexpected type for write value: %T\n", val)
	// 		returnWriteError(fmt.Sprintf("Unexpected type for write value: %T", val), &mqttResp)
	// 		return
	// 	}

	// 	variant, err := ua.NewVariant(writeReq.Value)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Failed to create new variant: %s\n", err.Error())
	// 		returnWriteError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	req := &ua.WriteRequest{
	// 		NodesToWrite: []*ua.WriteValue{
	// 			{
	// 				NodeID:      id,
	// 				AttributeID: ua.AttributeIDValue,
	// 				Value: &ua.DataValue{
	// 					EncodingMask: ua.DataValueValue,
	// 					Value:        variant,
	// 				},
	// 			},
	// 		},
	// 	}

	// 	resp, err := opcuaClient.Write(req)
	// 	if err != nil {
	// 		log.Printf("[ERROR] Failed to write OPC UA tag %s: %s\n", writeReq.NodeID, err.Error())
	// 		returnWriteError(err.Error(), &mqttResp)
	// 		return
	// 	}

	// 	if resp.Results[0] != ua.StatusOK {
	// 		log.Printf("[ERROR] non ok status returned from write: %s\n", resp.Results[0].Error())
	// 		mqttResp.StatusCode = uint32(resp.Results[0])
	// 		returnWriteError(fmt.Sprintf("Non OK status code returned from write: %s\n", resp.Results[0].Error()), &mqttResp)
	// 		return
	// 	}

	// 	mqttResp.NodeID = writeReq.NodeID
	// 	mqttResp.Timestamp = resp.ResponseHeader.Timestamp.UTC().Format(time.RFC3339)
	// 	mqttResp.StatusCode = uint32(resp.ResponseHeader.ServiceResult)

	// 	log.Printf("[INFO] OPC UA write successful: %+v\n", resp.Results[0])

	// 	publishJson(adapterConfig.TopicRoot+"/"+writeTopic+"/response", mqttResp)
}

func writeTag(tag *eip.Tag) {
	err := tag.Read()
	if err != nil {
		// cannot read tag
		log.Fatalln(err)
	}
}

// func getConvertedValue(nodeType *ua.TypeID, value interface{}) (interface{}, error) {
// 	switch *nodeType {
// 	case ua.TypeIDBoolean:
// 		return value.(bool), nil
// 	case ua.TypeIDDateTime:
// 		return value.(string), nil
// 	case ua.TypeIDDouble:
// 		return value.(float64), nil
// 	case ua.TypeIDFloat:
// 		return float32(value.(float64)), nil
// 	case ua.TypeIDGUID:
// 		return value.(string), nil
// 	case ua.TypeIDInt16:
// 		return int16(value.(float64)), nil
// 	case ua.TypeIDInt32:
// 		return int32(value.(float64)), nil
// 	case ua.TypeIDInt64:
// 		return int64(value.(float64)), nil
// 	case ua.TypeIDLocalizedText:
// 		return value.(string), nil
// 	case ua.TypeIDNodeID:
// 		return value.(string), nil
// 	case ua.TypeIDQualifiedName:
// 		return value.(string), nil
// 	case ua.TypeIDString:
// 		return value.(string), nil
// 	case ua.TypeIDUint16:
// 		return uint16(value.(float64)), nil
// 	case ua.TypeIDUint32:
// 		return uint32(value.(float64)), nil
// 	case ua.TypeIDUint64:
// 		return uint64(value.(float64)), nil
// 	case ua.TypeIDVariant:
// 		return value.(bool), nil
// 	case ua.TypeIDXMLElement:
// 		return value.(string), nil
// 	default:
// 		return nil, fmt.Errorf("Unhandled node type: " + nodeType.String())
// 	}
// }

// func getTagDataType(nodeid *ua.NodeID) (*ua.TypeID, error) {
// 	log.Printf("[INFO] getTagDataType - checking type for node id: %s\n", nodeid.String())

// 	req := &ua.ReadRequest{
// 		MaxAge:             2000,
// 		NodesToRead:        []*ua.ReadValueID{},
// 		TimestampsToReturn: ua.TimestampsToReturnBoth,
// 	}

// 	req.NodesToRead = append(req.NodesToRead, &ua.ReadValueID{
// 		NodeID: nodeid,
// 	})

// 	opcuaResp, err := opcuaClient.Read(req)
// 	if err != nil {
// 		log.Printf("[ERROR] Read type request failed: %s\n", err.Error())
// 		return nil, err
// 	}

// 	if opcuaResp.Results[0].Status != ua.StatusOK {
// 		return nil, fmt.Errorf("read type status not OK for node id %s: %+v", nodeid.String(), opcuaResp.Results[0].Status)
// 	}

// 	log.Printf("[INFO] getTagDataType - type for node id %s: %s\n", nodeid.String(), opcuaResp.Results[0].Value.Type().String())

// 	nodeType := opcuaResp.Results[0].Value.Type()

// 	return &nodeType, nil
// }

func returnReadError(errMsg string, resp *ethernetIpReadResponseMQTTMessage) {
	resp.Success = false
	resp.ErrorMessage = errMsg
	resp.ServerTimestamp = time.Now().UTC().Format(time.RFC3339)
	publishJson(adapterConfig.TopicRoot+"/"+readTopic+"/response", resp)
}

func returnWriteError(errMsg string, resp *ethernetIpWriteResponseMQTTMessage) {
	resp.Success = false
	resp.ErrorMessage = errMsg
	resp.Timestamp = time.Now().UTC().Format(time.RFC3339)
	publishJson(adapterConfig.TopicRoot+"/"+writeTopic+"/response", resp)
}

// Publishes data to a topic
func publishJson(topic string, data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ERROR] Failed to stringify JSON: %s\n", err.Error())
		return
	}

	log.Printf("[DEBUG] publish - Publishing to topic %s\n", topic)
	err = adapter_library.Publish(topic, b)
	if err != nil {
		log.Printf("[ERROR] Failed to publish MQTT message to topic %s: %s\n", topic, err.Error())
	}
}

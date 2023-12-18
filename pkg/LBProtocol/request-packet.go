package LBProtocol

import (
    "fmt"
    "errors"
    "github.com/google/gopacket"
	"encoding/json"
)

// Register the layer type
var LBLayerType = gopacket.RegisterLayerType(
    2001,
    gopacket.LayerTypeMetadata{
        "LBRequestLayer",
        gopacket.DecodeFunc(decodeLBRequest),
    },
)

type Mappings struct {
	Paths []string
	Address string
}

type LBPacket struct {
    Action bool
    Payload Mappings
}
// Create LBRequestLayer
type LBRequestLayer struct {
    Protocol [2]byte
    Action    byte
    RemainingLength byte
    Payload  []byte
}

func (l LBRequestLayer) LayerType() gopacket.LayerType {
    return LBLayerType
}

func (l LBRequestLayer) LayerContents() []byte {
    bytes := append([]byte{l.Action, l.RemainingLength}, l.Payload...)
    return append(l.Protocol[:], bytes...)
}

// LayerPayload returns nil as CustomLayerType is the only layer
func (l LBRequestLayer) LayerPayload() []byte {
    return nil
}

// Custom decode function.

func decodeLBRequest(data []byte, p gopacket.PacketBuilder) error {
    // AddLayer Layer
    p.AddLayer(
        &LBRequestLayer{
            [2]byte{data[0], data[1]}, 
            data[2], 
            data[3], 
            data[4:]})
    return nil
}

func (l LBRequestLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
    // Serialize the custom data into the buffer
    length := 4 + len(l.Payload)
    byteArray, err := b.AppendBytes(length)
    fmt.Println("byte: ", byteArray)
    fmt.Printf("bytearray type: %T\n", byteArray)
    if err != nil {
        return err
    }
    byteArray[0] = l.Protocol[0]
    byteArray[1] = l.Protocol[1]
    byteArray[2] = l.Action
    byteArray[3] = l.RemainingLength
    byteArray = append(byteArray[:4], l.Payload...)
    return err
}

func (l LBRequestLayer) Deserialize() (*LBPacket, error) {
    
    action := false
    if l.Action == 0xF0{
        action = true
    } else if l.Action == 0x0F{
        action = false
    } else {
        return nil, errors.New("Invalid Action")
    }

    fmt.Println("Inside deserialization, payload: ", l.Payload)
    var payload Mappings
    err := json.Unmarshal(l.Payload, &payload )
    if err != nil  {
        return nil, err
    }

    packet := LBPacket {
        Action: action,
        Payload: payload,
    }

    return &packet, nil

}
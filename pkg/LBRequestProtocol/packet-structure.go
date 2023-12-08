package LBRequestProtocol

import (
    "fmt"
    "github.com/google/gopacket"
	"encoding/json"
)

// Create LBRequestLayer
type LBRequestLayer struct {
    Protocol [2]byte
    Action    byte
    RemainingLength byte
    Payload  []byte
}

type Mappings struct {
	paths []string
	address string
}

// Register the layer type
var CustomLayerType = gopacket.RegisterLayerType(
    2001,
    gopacket.LayerTypeMetadata{
        "LBRequestLayer",
        gopacket.DecodeFunc(decodeLBRequest),
    },
)

func (l LBRequestLayer) LayerType() gopacket.LayerType {
    return CustomLayerType
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

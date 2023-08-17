package LBRequestProtocol

import (
    "fmt"
    "github.com/google/gopacket"
	"encoding/json"
)

// Create LBRequestLayer
type LBRequestLayer struct {
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
    return append([]byte{l.Action, l.RemainingLength}, l.Payload...)
}

// LayerPayload returns nil as CustomLayerType is the only layer
func (l LBRequestLayer) LayerPayload() []byte {
    return nil
}

// Custom decode function.

func decodeLBRequest(data []byte, p gopacket.PacketBuilder) error {
    // AddLayer Layer
    p.AddLayer(&LBRequestLayer{data[0], data[1], data[2:]})
    return nil
}

func (l LBRequestLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
    // Serialize the custom data into the buffer
    length := 2 + len(l.Payload)
    byteArray, err := b.AppendBytes(length)
    fmt.Println("byte: ", byteArray)
    fmt.Printf("bytearray type: %T\n", byteArray)
    if err != nil {
        return err
    }
    byteArray[0] = l.Action
    byteArray[1] = l.RemainingLength
    byteArray = append(byteArray[2:], l.Payload...)
    return err
}

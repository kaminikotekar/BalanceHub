package LBProtocol

import (
    "fmt"
    "github.com/google/gopacket"
)

// Register the layer type
var ErrLayerType = gopacket.RegisterLayerType(
    2002,
    gopacket.LayerTypeMetadata{
        "ErrLayer",
        gopacket.DecodeFunc(decodeErrRequest),
    },
)
// Create LBRequestLayer
type ErrLayer struct {
    Protocol [2]byte
	Result byte
    RemainingLength byte
    Payload  []byte
}

func (e ErrLayer) LayerType() gopacket.LayerType {
    return LBLayerType
}

func (e ErrLayer) LayerContents() []byte {
    bytes := append([]byte{e.Result, e.RemainingLength}, e.Payload...)
    return append(e.Protocol[:], bytes...)
}

// LayerPayload returns nil as CustomLayerType is the only layer
func (e ErrLayer) LayerPayload() []byte {
    return nil
}

// Custom decode function.

func decodeErrRequest(data []byte, p gopacket.PacketBuilder) error {
    // AddLayer Layer
    p.AddLayer(
        &ErrLayer{
            [2]byte{data[0], data[1]}, 
            data[2], 
			data[3], 
            data[4:]})
    return nil
}

func (e ErrLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
    // Serialize the custom data into the buffer
    length := 4 + len(e.Payload)
    byteArray, err := b.AppendBytes(length)
    fmt.Println("byte: ", byteArray)
    fmt.Printf("bytearray type: %T\n", byteArray)
    if err != nil {
        return err
    }
    byteArray[0] = e.Protocol[0]
    byteArray[1] = e.Protocol[1]
	byteArray[2] = e.Result
    byteArray[3] = e.RemainingLength
    byteArray = append(byteArray[:4], e.Payload...)
    return err
}

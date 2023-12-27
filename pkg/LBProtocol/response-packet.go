package LBProtocol

import (
    "fmt"
    "github.com/google/gopacket"
)

// Register the layer type
var RespLayerType = gopacket.RegisterLayerType(
    2002,
    gopacket.LayerTypeMetadata{
        "RespLayer",
        gopacket.DecodeFunc(decodeErrRequest),
    },
)
// Create LBRequestLayer
type RespLayer struct {
    Protocol [2]byte
	Result byte
    RemainingLength byte
    Payload  []byte
}

func (r RespLayer) LayerType() gopacket.LayerType {
    return LBLayerType
}

func (r RespLayer) LayerContents() []byte {
    bytes := append([]byte{r.Result, r.RemainingLength}, r.Payload...)
    return append(r.Protocol[:], bytes...)
}

// LayerPayload returns nil as CustomLayerType is the only layer
func (r RespLayer) LayerPayload() []byte {
    return nil
}

// Custom decode function.

func decodeErrRequest(data []byte, p gopacket.PacketBuilder) error {
    // AddLayer Layer
    p.AddLayer(
        &RespLayer{
            [2]byte{data[0], data[1]}, 
            data[2], 
			data[3], 
            data[4:]})
    return nil
}

func (r RespLayer) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
    // Serialize the custom data into the buffer
    length := 4 + len(r.Payload)
    byteArray, err := b.AppendBytes(length)
    fmt.Println("byte: ", byteArray)
    fmt.Printf("bytearray type: %T\n", byteArray)
    if err != nil {
        return err
    }
    byteArray[0] = r.Protocol[0]
    byteArray[1] = r.Protocol[1]
	byteArray[2] = r.Result
    byteArray[3] = r.RemainingLength
    byteArray = append(byteArray[:4], r.Payload...)
    return err
}

func GenerateResponse(result bool, payload string) []byte {

    rawBytes := []byte("RE")
	var message []byte
	if result == true {
		rawBytes = append(rawBytes, 0xFF)
	} else{
		rawBytes = append(rawBytes, 0x00)
	}
    message = []byte(payload)

	rawBytes = append(rawBytes, byte(len(message)))
	pData := append(rawBytes, message...)

    return pData
}

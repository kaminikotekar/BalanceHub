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


func Test(){
    // If you create your own encoding and decoding you can essentially
    // create your own protocol or implement a protocol that is not
    // already defined in the layers package. In our example we are just
    // wrapping a normal ethernet packet with our own layer.
    // Creating your own protocol is good if you want to create
    // some obfuscated binary data type that was difficult for others
    // to decode

    // Finally, decode your packets:
    // rawBytes := []byte{0xF0, 65, 65, 66, 67, 68}
    rawBytes := []byte("LB")
	rawBytes = append(rawBytes, 0xF0)


	payloadData := Mappings{
		Paths: []string{"/getList", "/PostList"},
		Address: "SomeAddress",
	}
	pydata ,_ := json.Marshal(payloadData)
	pData := append(rawBytes, pydata...)
    packet := gopacket.NewPacket(
        pData,
        LBLayerType,
        gopacket.Default,
    )

    fmt.Println("Created packet out of raw bytes.")
    fmt.Println(packet)
	fmt.Printf("Type packet %T \n", packet)

    // Decode the packet as our custom layer
    customLayer := packet.Layer(LBLayerType)
    if customLayer != nil {
        fmt.Println("Packet was successfully decoded with custom layer decoder.")
        customLayerContent, _ := customLayer.(*LBRequestLayer)
        // Now we can access the elements of the custom struct
        fmt.Println("Payload: ", customLayerContent.LayerPayload())
        fmt.Println("SomeByte element:", customLayerContent.Action)
        // fmt.Println("AnotherByte element:", customLayerContent.AnotherByte)
    }

    buf := gopacket.NewSerializeBuffer()
    opts := gopacket.SerializeOptions{} 


    // var bytesToSend []byte
	for _, layer := range packet.Layers() {
		fmt.Println("PACKET LAYER:", layer.LayerType())

	}

    err := gopacket.SerializePacket(buf, opts, packet)
    fmt.Println("err: ", err)
    fmt.Println("Packet bytes: ", buf.Bytes())
	// return packet
}
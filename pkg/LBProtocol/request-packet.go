package LBProtocol

import (
	"encoding/json"
	"errors"
	"github.com/google/gopacket"
	"github.com/kaminikotekar/BalanceHub/pkg/Connection"
	"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
	"log"
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
	Port    string
	Paths   []string
	Clients []string
}

type LBPacket struct {
	Action  bool
	Payload Mappings
}

// Create LBRequestLayer
type LBRequestLayer struct {
	Protocol        [2]byte
	Action          byte
	RemainingLength byte
	Payload         []byte
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
	if l.Action == 0xF0 {
		action = true
	} else if l.Action == 0x0F {
		action = false
	} else {
		return nil, errors.New("Invalid Action")
	}

	var payload Mappings
	err := json.Unmarshal(l.Payload, &payload)
	if err != nil {
		return nil, err
	}

	packet := LBPacket{
		Action:  action,
		Payload: payload,
	}

	return &packet, nil

}

func DecodeToPacket(buffer []byte) (*LBPacket, error) {
	remainingLength := buffer[3]
	packetLength := remainingLength + 4

	packet := gopacket.NewPacket(buffer[:packetLength],
		LBLayerType,
		gopacket.Default)

	customLayer := packet.Layer(LBLayerType)
	customLayerContent, _ := customLayer.(*LBRequestLayer)
	decodedPacket, err := customLayerContent.Deserialize()

	return decodedPacket, err
}

func (p *LBPacket) HandleRemoteRequest(remoteIP string, remotePort string) error {

	if p.Payload.Port != "" {
		remotePort = p.Payload.Port
	}

	RemoteID, err := Connection.HandleDBRequests(p.Action, remoteIP, remotePort, p.Payload.Paths, p.Payload.Clients)
	if err != nil {
		log.Println("error while handling remote request in Db ", err)
		return err
	}

	if p.Action == true {
		RemoteServer.RemoteServerMap.AddServer(RemoteID, remoteIP, remotePort)

		// Update path
		for _, path := range p.Payload.Paths {
			RemoteServer.RemoteServerMap.UpdatePath(path, RemoteID)
		}
		// Update Client
		for _, client := range p.Payload.Clients {
			RemoteServer.RemoteServerMap.UpdateClientIP(client, RemoteID)
		}
	} else {
		if len(p.Payload.Paths) == 0 && len(p.Payload.Clients) == 0 {
			RemoteServer.RemoteServerMap.RemoveServer(RemoteID)
			return nil
		}
		for _, path := range p.Payload.Paths {
			RemoteServer.RemoteServerMap.DeletePath(path, RemoteID)
		}
		// Update Client
		for _, client := range p.Payload.Clients {
			RemoteServer.RemoteServerMap.DeleteClient(client, RemoteID)
		}
	}

	return nil
}

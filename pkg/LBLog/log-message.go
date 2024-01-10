package LBLog

import (
	"encoding/json"
	"log"
	"time"
)

type Message struct {
	Message   string
	Timestamp string
	Flag      string
}

func (msg *Message) marshal() []byte {
	encoded, _ := json.Marshal(msg)
	return encoded
}

func unMarshal(rawMsg []byte) *Message {
	var msg Message
	json.Unmarshal(rawMsg, &msg)
	return &msg
}

func Log(flag string, message string) {

	if !Initialized {
		log.Println(WARNING, "Log worker not initialized !!")
		return
	}
	msg := &Message{
		Message:   message,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Flag:      flag,
	}
	if lbLogger.isRDLogger {
		WriteRDMQ(ctx, client, msg.marshal())
		return
	}
	messageQueue <- msg
}

package services

import (
	"../logger"
	"../material"
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
)

type Message material.Message // just simplify the access to the Message struct

type connectionHandler struct {
	connection       *net.Conn
	connectionClosed bool
	registeredTopics []string
	RegisterEvent    []func(connectionHandler, []string) // will be fired when a client registeres himself at some topics
	UnregisterEvent  []func(connectionHandler, []string) // will be fired when a client un-registeres himself at some topics
	SendEvent        []func(connectionHandler, []string, string)
}

func (ch *connectionHandler) Init(connection *net.Conn) {
	ch.connection = connection
}

func (ch *connectionHandler) HandleConnection() {
	if ch.connection == nil {
		logger.Error("Connection not set!")
		return
	}

	ch.waitFor(
		[]string{material.MtRegister},
		[]func(Message){ch.handleRegistration})

	for true {
		ch.waitFor(
			[]string{material.MtRegister,
				material.MtLogout,
				material.MtClose,
				material.MtSend},
			[]func(Message){ch.handleRegistration,
				ch.handleLogout,
				ch.handleClose,
				ch.handleSending})

		if ch.connectionClosed {
			break
		}
	}
}

func (ch *connectionHandler) waitFor(messageTypes []string, handler []func(message Message)) {
	rawMessage, err := bufio.NewReader(*ch.connection).ReadString('\n')

	if err == nil {
		// the length of the message that should be printed
		maxOutputLength := int(math.Min(float64(len(rawMessage))-1, 30))
		output := rawMessage[:maxOutputLength]
		if 30 < len(rawMessage)-1 {
			output += " [...]"
		}
		logger.Info(output)

		// JSON to Message-struct
		message := ch.getMessageFromJSON(rawMessage)

		// check type
		for i := 0; i < len(messageTypes); i++ {
			messageType := messageTypes[i]
			logger.Info("Check " + messageType + " type")

			if message.MessageType == messageType {
				logger.Info("Handle " + messageType + " type")
				handler[i](message)
				break
			}
		}
	}
}

func (ch *connectionHandler) getMessageFromJSON(jsonData string) Message {
	message := Message{}
	json.Unmarshal([]byte(jsonData), &message)
	return message
}

func (ch *connectionHandler) handleRegistration(message Message) {
	logger.Debug("Register to topics " + fmt.Sprintf("%#v", message.Topics))

	for _, event := range ch.RegisterEvent {
		event(*ch, message.Topics)
	}

	for _, topic := range message.Topics {
		if !contains(ch.registeredTopics, topic) {
			ch.registeredTopics = append(ch.registeredTopics, topic)
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (ch *connectionHandler) handleSending(message Message) {
	logger.Error("NOT IMPLEMENTED!")
	logger.Debug(fmt.Sprintf("Send message to topics %#v", message.Topics))
}

func (ch *connectionHandler) handleLogout(message Message) {
	logger.Debug(fmt.Sprintf("Unsubscribe from topics %#v", message.Topics))
	ch.logout(message.Topics)
}

func (ch *connectionHandler) handleClose(message Message) {
	logger.Debug("Unsubscribe from all topics")
	ch.logout(ch.registeredTopics)

	logger.Debug("Close connection")
	(*ch.connection).Close()
	ch.connectionClosed = true
}

func (ch *connectionHandler) logout(topics []string) {
	for _, event := range ch.UnregisterEvent {
		event(*ch, topics)
	}
}

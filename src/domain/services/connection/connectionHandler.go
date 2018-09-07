package connectionServices

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/go-messaging-service/goms-server/src/domain/material"
	"github.com/go-messaging-service/goms-server/src/domain/services/common"
	"github.com/go-messaging-service/goms-server/src/technical/common"
	"github.com/go-messaging-service/goms-server/src/technical/material"
	"github.com/hauke96/sigolo"
)

type Message material.Message // just simplify the access to the Message struct

type ConnectionHandler struct {
	Connection       *net.Conn
	connectionClosed bool
	config           *technicalMaterial.Config
	registeredTopics []string
	SendEvent        []func(ConnectionHandler, *Message)
}

const MAX_PRINTING_LENGTH int = 80

// Init initializes the handler with the given connection.
func (ch *ConnectionHandler) Init(connection *net.Conn, config *technicalMaterial.Config) {
	ch.Connection = connection
	ch.config = config
}

// HandleConnection starts a routine to handle registration and sending messages.
// This will run until the client logs out, so run this in a goroutine.
func (ch *ConnectionHandler) HandleConnection() {
	// Not initialized
	if ch.Connection == nil {
		sigolo.Fatal("Connection not set!")
	}

	messageTypes := []string{material.MT_REGISTER,
		material.MT_LOGOUT,
		material.MT_CLOSE,
		material.MT_SEND}

	handler := []func(Message){ch.handleRegistration,
		ch.handleLogout,
		ch.handleClose,
		ch.handleSending}

	reader := bufio.NewReader(*ch.Connection)

	// Now a arbitrary amount of registration, logout, close and send messages is allowed
	for !ch.connectionClosed {
		ch.waitFor(
			messageTypes,
			handler,
			reader)
	}
}

// waitFor wats until on of the given message types arrived.
// The i-th argument in the messageTypes array must match to the i-th argument in the handler array.
func (ch *ConnectionHandler) waitFor(messageTypes []string, handler []func(message Message), reader *bufio.Reader) {

	// Check if the arrays match and error/fatal here
	if len(messageTypes) != len(handler) {
		if len(messageTypes) > len(handler) {
			// Fatal here to prevent a "slice bounds out of range" error during runtime
			sigolo.Fatal("There're more defined message types then functions mapped to them.")
		} else {
			sigolo.Error("There're more defined functions then message types here. Some message types might not be covered. Fix that!")
		}
	}

	rawMessage, err := reader.ReadString('\n')

	if err == nil {
		// the length of the message that should be printed
		maxOutputLength := int(math.Min(float64(len(rawMessage))-1, float64(MAX_PRINTING_LENGTH)))
		output := rawMessage[:maxOutputLength]
		if MAX_PRINTING_LENGTH < len(rawMessage)-1 {
			output += " [...]"
		}
		sigolo.Info(output)

		// JSON to Message-struct
		message := getMessageFromJSON(rawMessage)

		// check type
		for i := 0; i < len(messageTypes); i++ {
			messageType := messageTypes[i]
			sigolo.Debug("Check if received type '" + message.Messagetype + "' is type '" + messageType + "'")

			if message.Messagetype == messageType {
				sigolo.Debug("Handle " + messageType + " type")
				handler[i](message)
				break
			}
		}
	} else {
		sigolo.Info("The connection will be closed. Reason: " + err.Error())
		ch.exit()
		ch.connectionClosed = true
	}
}

// getMessageFromJSON converts the given json-data into a message object.
func getMessageFromJSON(jsonData string) Message {
	message := Message{}
	json.Unmarshal([]byte(jsonData), &message)
	return message
}

// handleRegistration registeres this connection to the topics specified in the message.
func (ch *ConnectionHandler) handleRegistration(message Message) {
	sigolo.Debug("Try to register to topics " + fmt.Sprintf("%#v", message.Topics))

	// A comma separated list of all topics, the client is not allowed to register to
	forbiddenTopics := ""
	alreadyRegisteredTopics := ""

	for _, topic := range message.Topics {
		//TODO create a service for this. This should later take care of different user rights
		if !technicalCommon.ContainsString(ch.config.TopicConfig.Topics, topic) {
			forbiddenTopics += topic + ","
			sigolo.Info("Clients wants to register on invalid topic (" + topic + ").")

		} else if technicalCommon.ContainsString(ch.registeredTopics, topic) {
			alreadyRegisteredTopics += topic + ","
			sigolo.Debug("Client already registered on " + topic)

		} else {
			ch.registeredTopics = append(ch.registeredTopics, topic)
			sigolo.Debug("Register " + topic)
		}
	}

	// Send error message for forbidden topics and cut trailing comma
	if len(forbiddenTopics) != 0 {
		forbiddenTopics = strings.TrimSuffix(forbiddenTopics, ",")
		commonServices.SendErrorMessage(ch.Connection, material.ERR_REG_INVALID_TOPIC, forbiddenTopics)
	}

	// Send error message for already registered topics and cut trailing comma
	if len(alreadyRegisteredTopics) != 0 {
		alreadyRegisteredTopics = strings.TrimSuffix(alreadyRegisteredTopics, ",")
		commonServices.SendErrorMessage(ch.Connection, material.ERR_REG_ALREADY_REGISTERED, alreadyRegisteredTopics)
	}
}

// handleSending send the given message to all clients interested in the topics specified in the message.
func (ch *ConnectionHandler) handleSending(message Message) {
	for _, event := range ch.SendEvent {
		event(*ch, &message)
	}
}

// handleLogout logs the client out.
func (ch *ConnectionHandler) handleLogout(message Message) {
	sigolo.Debug(fmt.Sprintf("Unsubscribe from topics %#v", message.Topics))
	ch.logout(message.Topics)
}

// handleClose logs the client out from all topics and closes the connection.
func (ch *ConnectionHandler) handleClose(message Message) {
	ch.exit()
}

// exit logs the client out from all topics and closes the connection.
func (ch *ConnectionHandler) exit() {
	sigolo.Debug("Unsubscribe from all topics")
	ch.logout(ch.registeredTopics)

	sigolo.Debug("Close connection")
	(*ch.Connection).Close()
	ch.connectionClosed = true
}

// logout will logs the client out from the given topics.
func (ch *ConnectionHandler) logout(topics []string) {
	for _, topic := range topics {
		ch.registeredTopics = technicalCommon.RemoveString(ch.registeredTopics, topic)
	}

	ch.registeredTopics = technicalCommon.RemoveStrings(ch.registeredTopics, topics)
}

func (ch *ConnectionHandler) IsRegisteredTo(topic string) bool {
	return technicalCommon.ContainsString(ch.registeredTopics, topic)
}

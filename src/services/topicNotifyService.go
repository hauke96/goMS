package services

import (
	domain "goMS/src/material"
	technical "goMS/src/technical/material"
	"goMS/src/technical/services/logger"
)

type Notification technical.Notification

type TopicNotifyService struct {
	queue       chan *Notification
	errors      chan *Notification
	exit        chan bool
	initialized bool
}

func (tn *TopicNotifyService) Init() {
	tn.queue = make(chan *Notification)
	tn.errors = make(chan *Notification)
	tn.exit = make(chan bool)

	tn.initialized = true
}

func (tn *TopicNotifyService) StartNotifier() {
	// Not initialized
	if !tn.initialized {
		logger.Error("TopicNotifyService not initialized")
	}

	for {
		select {
		case notification := <-tn.queue:
			tn.sendNotification(notification)
		case <-tn.exit:
			return
		}
	}
}

func (tn *TopicNotifyService) sendNotification(notification *Notification) {
	for _, connection := range *notification.Connections {

		err := sendMessageTo(connection, notification.Data)

		if err != nil {
			sendErrorMessage(connection, domain.ERR_SEND_FAILED, err.Error())
		}
	}
}
package transport

import "2sp/pkg/protocol"

type Service interface {
	SubscribeToMessages(room *protocol.Room) (*MessagesSubscription, error)
	PublishUnencryptedMessage(room *protocol.Room, payload []byte) error
	PublishPublicMessage(room *protocol.Room, payload []byte) error
	PublishPrivateMessage(room *protocol.Room, payload []byte) error
}

type MessagesSubscription struct {
	Ch          chan []byte
	Unsubscribe func()
}
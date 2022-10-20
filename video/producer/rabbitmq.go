package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	myRabbitmq "nkonev.name/video/rabbitmq"
	"nkonev.name/video/utils"
	"time"
)

const videoDialStatusQueue = "video-dial-statuses"
const AsyncEventsFanoutExchange = "async-events-exchange"

func (rp *RabbitNotificationsPublisher) Publish(participantIds []int64, chatNotifyDto *dto.VideoCallChangedDto, ctx context.Context) error {

	for _, participantId := range participantIds {
		event := dto.GlobalEvent{
			EventType:         "video_call_changed",
			UserId:            participantId,
			VideoNotification: chatNotifyDto,
		}

		bytea, err := json.Marshal(event)
		if err != nil {
			GetLogEntry(ctx).Error(err, "Failed during marshal chatNotifyDto")
			continue
		}

		msg := amqp.Publishing{
			DeliveryMode: amqp.Transient,
			Timestamp:    time.Now(),
			ContentType:  "application/json",
			Body:         bytea,
			Type:         utils.GetType(event),
		}

		if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
			GetLogEntry(ctx).Error(err, "Error during publishing")
			continue
		}
	}
	return nil
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitNotificationsPublisher(connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}

func (rp *RabbitInvitePublisher) Publish(invitationDto *dto.VideoCallInvitation, toUserId int64) error {
	event := dto.GlobalEvent{
		EventType:           "video_call_invitation",
		UserId:              toUserId,
		VideoChatInvitation: invitationDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		Logger.Error(err, "Failed during marshal videoChatInvitationDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing")
	}
	return err
}

type RabbitInvitePublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitInvitePublisher(connection *rabbitmq.Connection) *RabbitInvitePublisher {
	return &RabbitInvitePublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}

func (rp *RabbitDialStatusPublisher) Publish(req *dto.VideoIsInvitingDto) error {
	var dials = []*dto.VideoDialChanged{}
	for _, userId := range req.UserIds {
		dials = append(dials, &dto.VideoDialChanged{
			UserId: userId,
			Status: req.Status,
		})
	}

	event := dto.GlobalEvent{
		EventType: "video_dial_status_changed",
		UserId:    req.BehalfUserId,
		VideoParticipantDialEvent: &dto.VideoDialChanges{
			ChatId: req.ChatId,
			Dials:  dials,
		},
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		Logger.Error(err, "Failed during marshal videoChatInvitationDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing")
	}
	return err

}

type RabbitDialStatusPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitDialStatusPublisher(connection *rabbitmq.Connection) *RabbitDialStatusPublisher {
	return &RabbitDialStatusPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}

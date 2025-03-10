package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/video/dto"
	"nkonev.name/video/logger"
	myRabbitmq "nkonev.name/video/rabbitmq"
	"nkonev.name/video/utils"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"
const NotificationsFanoutExchange = "notifications-exchange"

func (rp *RabbitUserCountPublisher) Publish(ctx context.Context, participantIds []int64, chatNotifyDto *dto.VideoCallUserCountChangedDto) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	for _, participantId := range participantIds {
		event := dto.GlobalUserEvent{
			EventType:               "video_user_count_changed",
			UserId:                  participantId,
			VideoCallUserCountEvent: chatNotifyDto,
		}

		bytea, err := json.Marshal(event)
		if err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal chatNotifyDto")
			continue
		}

		msg := amqp.Publishing{
			DeliveryMode: amqp.Transient,
			Timestamp:    time.Now().UTC(),
			ContentType:  "application/json",
			Body:         bytea,
			Type:         utils.GetType(event),
			Headers:      headers,
		}

		if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
			continue
		}
	}
	return nil
}

type RabbitUserCountPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitUserCountPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitUserCountPublisher {
	return &RabbitUserCountPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

// in_video = red dot
func (rp *RabbitUserIdsPublisher) Publish(ctx context.Context, videoCallUsersCallStatusChanged *dto.VideoCallUsersCallStatusChangedDto) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	event := dto.GeneralEvent{
		EventType:                            "user_in_video_call_changed",
		VideoCallUsersCallStatusChangedEvent: videoCallUsersCallStatusChanged,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal chatNotifyDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
		Headers:      headers,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
		return err
	}
	return nil
}

type RabbitUserIdsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitUserIdsPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitUserIdsPublisher {
	return &RabbitUserIdsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

// sends "you are invited"
func (rp *RabbitInvitePublisher) Publish(ctx context.Context, invitationDto *dto.VideoCallInvitation, toUserId int64) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	event := dto.GlobalUserEvent{
		EventType:           "video_call_invitation",
		UserId:              toUserId,
		VideoChatInvitation: invitationDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal videoChatInvitationDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
		Headers:      headers,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
	}
	return err
}

type RabbitInvitePublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitInvitePublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitInvitePublisher {
	return &RabbitInvitePublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

// send info about user being invited, used in ChatParticipants (blinking green tube) and ChatView (blinking it tet-a-tet)
func (rp *RabbitDialStatusPublisher) Publish(
	ctx context.Context,
	chatId int64,
	userStatuses map[int64]string,
	toUserId int64,
) {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	if len(userStatuses) == 0 {
		return
	}
	var dials = []*dto.VideoDialChanged{}
	for userId, status := range userStatuses {
		dials = append(dials, &dto.VideoDialChanged{
			UserId: userId,
			Status: status,
		})
	}

	event := dto.GlobalUserEvent{
		EventType: "video_dial_status_changed",
		UserId:    toUserId,
		VideoParticipantDialEvent: &dto.VideoDialChanges{
			ChatId: chatId,
			Dials:  dials,
		},
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal videoChatInvitationDto")
		return
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
		Headers:      headers,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
	}
}

type RabbitDialStatusPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitDialStatusPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitDialStatusPublisher {
	return &RabbitDialStatusPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

func (rp *RabbitRecordingPublisher) Publish(ctx context.Context, recordInProgressByOwner map[int64]bool, chatId int64) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	for participantId, recordInProgress := range recordInProgressByOwner {
		var chatNotifyDto = dto.VideoCallRecordingChangedDto{
			RecordInProgress: recordInProgress,
			ChatId:           chatId,
		}

		event := dto.GlobalUserEvent{
			EventType:               "video_recording_changed",
			UserId:                  participantId,
			VideoCallRecordingEvent: &chatNotifyDto,
		}

		bytea, err := json.Marshal(event)
		if err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal chatNotifyDto")
			continue
		}

		msg := amqp.Publishing{
			DeliveryMode: amqp.Transient,
			Timestamp:    time.Now().UTC(),
			ContentType:  "application/json",
			Body:         bytea,
			Type:         utils.GetType(event),
			Headers:      headers,
		}

		if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
			continue
		}
	}
	return nil
}

type RabbitRecordingPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitRecordingPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitRecordingPublisher {
	return &RabbitRecordingPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

func (rp *RabbitNotificationsPublisher) Publish(ctx context.Context, notification dto.NotificationEvent) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	bytea, err := json.Marshal(notification)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Headers:      headers,
	}

	if err := rp.channel.Publish(NotificationsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitNotificationsPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

func (rp *RabbitScreenSharePublisher) Publish(ctx context.Context, participantIds []int64, chatNotifyDto *dto.VideoCallScreenShareChangedDto) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	for _, participantId := range participantIds {
		event := dto.GlobalUserEvent{
			EventType:                      "video_screenshare_changed",
			UserId:                         participantId,
			VideoCallScreenShareChangedDto: chatNotifyDto,
		}

		bytea, err := json.Marshal(event)
		if err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal chatNotifyDto")
			continue
		}

		msg := amqp.Publishing{
			DeliveryMode: amqp.Transient,
			Timestamp:    time.Now().UTC(),
			ContentType:  "application/json",
			Body:         bytea,
			Type:         utils.GetType(event),
			Headers:      headers,
		}

		if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
			rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
			continue
		}
	}
	return nil
}

type RabbitScreenSharePublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitScreenSharePublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitScreenSharePublisher {
	return &RabbitScreenSharePublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

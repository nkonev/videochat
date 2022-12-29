package listener

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/services"
)

type MinioEventsListener func(*amqp.Delivery) error

func CreateMinioEventsListener(service *services.PreviewService) MinioEventsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		Logger.Debugf("Received %v", strData)

		var minioEvent = new(dto.MinioEvent)
		err := json.Unmarshal(bytesData, minioEvent)
		if err != nil {
			Logger.Errorf("Unable to unmarshall %v to MinioEevnt", strData)
			return err
		} else {
			service.HandleMinioEvent(minioEvent)
			return nil
		}
	}
}

package services

import (
	"context"
	"nkonev.name/storage/dto"
)

type ConvertingService struct {

}

func NewConvertingService() *ConvertingService {
	return &ConvertingService{

	}
}

func (s *ConvertingService) HandleConvertedEvent(ctx context.Context, event *dto.MinioEvent) {
	// download the original recording_123.webm to the tmp dir (configurable)
	// run ffmpeg
	// set tag recording=true in order to correct work utils.GetEventType in minio_listener
	// put recording_123_converted.webm to minio
	// rm recording_123_converted.webm from the temporary directory
}

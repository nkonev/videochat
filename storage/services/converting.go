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

func (s *ConvertingService) HandleConvertedEvent(ctx context.Context, event *dto.MinioEvent) *ConvertedResponse {
	return &ConvertedResponse{

	}
}

func (s *ConvertingService) SendToOwner(ctx context.Context, minioEvent *dto.MinioEvent, ownerId int64, response *ConvertedResponse) {
	if response != nil {

	}
}

type ConvertedResponse struct {

}

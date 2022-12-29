package services

import (
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
)

type PreviewService struct {
}

func NewPreviewService() *PreviewService {
	return &PreviewService{}
}

func (s PreviewService) HandleMinioEvent(data *dto.MinioEvent) {
	Logger.Infof("Got %v", data)
}

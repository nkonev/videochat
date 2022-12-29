package services

import (
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strings"
)

type PreviewService struct {
}

func NewPreviewService() *PreviewService {
	return &PreviewService{}
}

func (s PreviewService) HandleMinioEvent(data *dto.MinioEvent) {
	Logger.Debugf("Got %v", data)
	if strings.HasPrefix(data.EventName, utils.ObjectCreated) {
		if utils.IsImage(data.Key) {
			// TODO image preview
		} else if utils.IsVideo(data.Key) {
			// TODO video preview
		}
	} else if strings.HasPrefix(data.EventName, utils.ObjectRemoved) {
		// TODO remove the preview
	}
}

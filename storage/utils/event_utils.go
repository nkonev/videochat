package utils

import "errors"

// actually it is output event
type EventType int
const (
	UNKNOWN EventType = iota
	FILE_CREATED
	FILE_DELETED
	FILE_UPDATED
)

func GetEventType(eventName string) (EventType, error) {
	var eventType EventType
	switch eventName {
	case ObjectCreatedCompleteMultipartUpload:
		eventType = FILE_CREATED
	case ObjectRemovedDelete:
		eventType = FILE_DELETED
	case ObjectCreatedPutTagging:
		eventType = FILE_UPDATED
	case ObjectCreatedPut:
		eventType = FILE_UPDATED
	default:
		return UNKNOWN, errors.New("Unable to determine event type")
	}
	return eventType, nil
}

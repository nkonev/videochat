package dto

import "errors"

// https://min.io/docs/minio/linux/reference/minio-mc/mc-event-add.html#mc-event-supported-events

const ObjectCreated = "s3:ObjectCreated"
const ObjectRemoved = "s3:ObjectRemoved"

const ObjectCreatedCompleteMultipartUpload = ObjectCreated + ":CompleteMultipartUpload"

const ObjectRemovedDelete = ObjectRemoved + ":Delete"

const ObjectCreatedPutTagging = ObjectCreated + ":PutTagging"
const ObjectCreatedPut = ObjectCreated + ":Put"

// actually it is output event
type EventType int

const (
	UNKNOWN EventType = iota
	FILE_CREATED
	FILE_DELETED
	FILE_UPDATED
)

func GetEventType(eventName string, isRecording bool) (EventType, error) {
	var eventType EventType
	switch eventName {
	case ObjectCreatedCompleteMultipartUpload:
		eventType = FILE_CREATED
	case ObjectRemovedDelete:
		eventType = FILE_DELETED
	case ObjectCreatedPutTagging:
		eventType = FILE_UPDATED
	case ObjectCreatedPut:
		{
			if isRecording {
				eventType = FILE_CREATED
			} else {
				eventType = FILE_UPDATED
			}
		}
	default:
		return UNKNOWN, errors.New("Unable to determine event type")
	}
	return eventType, nil
}

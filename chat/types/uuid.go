package types

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	. "nkonev.name/chat/logger"
)

//
// Most common scalars
//

type UUID uuid.UUID

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (y *UUID) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("UUID must be a string")
	}

	parsed, err := uuid.Parse(str)

	if err != nil {
		Logger.Errorf("Error during unmarshalling uuid %v", err)
		return err
	}

	var va UUID = UUID(parsed)
	*y = va

	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (y UUID) MarshalGQL(w io.Writer) {
	var va = uuid.UUID(y)
	_, err := fmt.Fprintf(w, "%v", va)
	if err != nil {
		Logger.Errorf("Error during marshalling uuid %v", err)
	}
}

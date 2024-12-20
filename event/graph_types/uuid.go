package graph_types

import (
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"io"
	"nkonev.name/event/logger"
	"strings"
)

//
// Most common scalars
//

var lgr = logger.NewLogger()

// UnmarshalGQL implements the graphql.Unmarshaler interface
func UnmarshalUUID(v interface{}) (*uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		withoutDoubleQuotes := strings.ReplaceAll(v, "\"", "")
		parsed, err := uuid.Parse(withoutDoubleQuotes)
		if err != nil {
			lgr.Errorf("Error during unmarshalling uuid %v", err)
			return nil, err
		}
		return &parsed, nil
	default:
		return nil, fmt.Errorf("%T is not a uuid or string", v)
	}
}

// MarshalGQL implements the graphql.Marshaler interface
func MarshalUUID(u *uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := fmt.Fprintf(w, "\"%v\"", u)
		if err != nil {
			lgr.Errorf("Error during marshalling uuid %v", err)
		}
	})
}

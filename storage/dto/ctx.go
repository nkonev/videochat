package dto

import "context"

type ctxKeyUpploadIdType string

const ctxKeyUpploadId ctxKeyUpploadIdType = "ctxKeyUpploadId"

type ctxKeyPartNumberType string

const ctxKeyPartNumber ctxKeyPartNumberType = "ctxKeyPartNumber"

type ctxKeyRequestUserMetadataType string

const ctxKeyRequestUserMetadata ctxKeyRequestUserMetadataType = "ctxKeyRequestUserMetadata"

func ContextWithUpploadId(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyUpploadId, v)
}

func ContextWithPartNumber(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyPartNumber, v)
}

func ContextWithRequestUserMetadata(ctx context.Context, v bool) context.Context {
	return context.WithValue(ctx, ctxKeyRequestUserMetadata, v)
}

func UpploadIdFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(ctxKeyUpploadId).(string)
	return user, ok
}

func PartNumberFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(ctxKeyPartNumber).(string)
	return user, ok
}

func RequestUserMetadataFromContext(ctx context.Context) (bool, bool) {
	r, ok := ctx.Value(ctxKeyRequestUserMetadata).(bool)
	return r, ok
}
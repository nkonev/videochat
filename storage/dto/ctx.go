package dto

import "context"

type ctxKeyUpploadIdType string

const ctxKeyUpploadId ctxKeyUpploadIdType = "ctxKeyUpploadId"

type ctxKeyPartNumberType string

const ctxKeyPartNumber ctxKeyPartNumberType = "ctxKeyPartNumber"

type ctxKeyRequestUserMetadataType string

const ctxKeyRequestUserMetadata ctxKeyRequestUserMetadataType = "ctxKeyRequestUserMetadata"

type ctxKeyCacheResponseToStringType string

const ctxKeyCacheResponseToString ctxKeyCacheResponseToStringType = "ctxKeyCacheResponseToString"

func ContextWithUpploadId(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyUpploadId, v)
}

func ContextWithPartNumber(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyPartNumber, v)
}

func ContextWithRequestUserMetadata(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyRequestUserMetadata, true)
}

func ContextWithCacheResponseToString(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyCacheResponseToString, true)
}

func UpploadIdFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(ctxKeyUpploadId).(string)
	return user, ok
}

func PartNumberFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(ctxKeyPartNumber).(string)
	return user, ok
}

func RequestUserMetadataFromContext(ctx context.Context) bool {
	r, ok := ctx.Value(ctxKeyRequestUserMetadata).(bool)
	return r && ok
}

func CacheResponseToStringFromContext(ctx context.Context) bool {
	r, ok := ctx.Value(ctxKeyCacheResponseToString).(bool)
	return r && ok
}

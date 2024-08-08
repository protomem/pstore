package blobstore

import (
	"bytes"
	"context"
	"errors"
	"io"
)

var ErrObjectNotFound = errors.New("object not found")

type Object struct {
	Body io.ReadCloser
	Meta map[string]string
}

func NewObject(body io.ReadCloser) Object {
	return Object{Body: body, Meta: make(map[string]string)}
}

func emptyObject() Object {
	return NewObject(io.NopCloser(&bytes.Buffer{}))
}

func (o *Object) Set(key string, value string) {
	if o.Meta == nil {
		o.Meta = make(map[string]string)
	}
	o.Meta[key] = value
}

func (o *Object) Get(key string) (string, bool) {
	value, ok := o.Meta[key]
	return value, ok
}

type Storage interface {
	Get(ctx context.Context, parent, name string) (Object, error)
	Set(ctx context.Context, parent, name string, obj Object) error
	Put(ctx context.Context, parent, name string, obj Object) error
	Delete(ctx context.Context, parent, name string) error

	Close(ctx context.Context) error
}

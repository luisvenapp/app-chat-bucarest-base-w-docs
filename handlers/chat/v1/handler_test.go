package chatv1handler

import (
	"testing"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
)

func TestRequestValidation(t *testing.T) {
	requests := []proto.Message{}
	for _, request := range requests {
		if err := protovalidate.Validate(request); err != nil {
			t.Fatalf(`[validation error] %q, got: %v`, request, err)
		}
	}
}

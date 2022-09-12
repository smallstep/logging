package grpclog

import (
	"fmt"

	"go.uber.org/zap/zapcore"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	// PROTOJSONMarshaller is the marshaller used for serializing protobuf
	// messages. If needed, this variable can be reassigned with a different
	// marshaller with the same Marshal() signature.
	PROTOJSONMarshaller = protojson.MarshalOptions{}
)

type protojsonObjectMarshaler struct {
	pb proto.Message
}

func (j *protojsonObjectMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddReflected("msg", j)
}

func (j *protojsonObjectMarshaler) MarshalJSON() ([]byte, error) {
	b, err := PROTOJSONMarshaller.Marshal(j.pb)
	if err != nil {
		return nil, fmt.Errorf("grpclog: protojson serializer failed: %w", err)
	}
	return b, nil
}

package grpclog

import (
	"fmt"

	"go.uber.org/zap/zapcore"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	// JSONPbMarshaller is the marshaller used for serializing protobuf
	// messages. If needed, this variable can be reassigned with a different
	// marshaller with the same Marshal() signature.
	JSONPbMarshaller = protojson.MarshalOptions{}
)

type jsonpbObjectMarshaler struct {
	pb proto.Message
}

func (j *jsonpbObjectMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddReflected("msg", j)
}

func (j *jsonpbObjectMarshaler) MarshalJSON() ([]byte, error) {
	b, err := JSONPbMarshaller.Marshal(j.pb)
	if err != nil {
		return nil, fmt.Errorf("grpclog: jsonpb serializer failed: %w", err)
	}
	return b, nil
}

package grpclog

import (
	"bytes"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/logging"
)

var (
	// JsonPbMarshaller is the marshaller used for serializing protobuf
	// messages. If needed, this variable can be reassigned with a different
	// marshaller with the same Marshal() signature.
	JsonPbMarshaller grpc_logging.JsonPbMarshaler = &jsonpb.Marshaler{}
)

type jsonpbObjectMarshaler struct {
	pb proto.Message
}

func (j *jsonpbObjectMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddReflected("msg", j)
}

func (j *jsonpbObjectMarshaler) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	if err := JsonPbMarshaller.Marshal(b, j.pb); err != nil {
		return nil, fmt.Errorf("grpclog: jsonpb serializer failed: %v", err)
	}
	return b.Bytes(), nil
}

func marshalMessageJSON(key string, msg interface{}) []zap.Field {
	if p, ok := msg.(proto.Message); ok {
		return []zap.Field{
			zap.Object(key, &jsonpbObjectMarshaler{pb: p}),
		}
	}
	return nil
}

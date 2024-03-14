package grpclog

import (
	"context"
	"crypto/tls"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	panoramix "github.com/smallstep/panoramix/v4/requestid"

	"github.com/smallstep/logging"
	"github.com/smallstep/logging/requestid"
)

type serverLogger struct {
	*logging.Logger
	interceptorType interceptorType
	name            string
	logRequests     bool
	logResponses    bool
	timeFormat      string
}

func newServerLogger(logger *logging.Logger, typ interceptorType) *serverLogger {
	return &serverLogger{
		Logger:          logger,
		interceptorType: typ,
		name:            logger.Name(),
		logRequests:     logger.LogRequests(),
		logResponses:    logger.LogResponses(),
		timeFormat:      logger.TimeFormat(),
	}
}

func (l *serverLogger) Log(ctx context.Context, fullMethod string, t time.Time, duration time.Duration, extra []zap.Field, grpcErr error) {
	var pkg string
	code := status.Code(grpcErr)
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)
	parts := strings.Split(service, ".")
	if l := len(parts); l > 1 {
		pkg = strings.Join(parts[:l-1], ".")
		service = parts[l-1]
	}

	var name, requestID, tracingID string

	// Use (reflected) request ID for logging. It _could_ be empty if it wasn't set
	// by some (external) middleware, but we stil log the legacy request ID too, so
	// it shouldn't be too big of an issue.
	requestID = panoramix.FromContext(ctx)
	if requestID == "" {
		requestID = requestid.FromContext(ctx)
	}

	if s, ok := logging.GetName(ctx); ok {
		name = s
	} else {
		name = l.name
	}
	if tp, ok := logging.GetTraceparent(ctx); ok {
		tracingID = tp.String()
		if requestID == "" {
			requestID = tp.TraceID()
		}
	}

	fields := []zap.Field{
		zap.String("name", name),
		zap.String("system", "grpc"),
		zap.String("span.kind", "server"),
		zap.String("grpc.package", pkg),
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
		zap.String("grpc.code", code.String()),
		zap.String("request-id", requestID),
		zap.String("tracing-id", tracingID),
		zap.String("time", t.Format(l.timeFormat)),
		zap.Duration("durations", duration),
		zap.Int64("duration-ns", duration.Nanoseconds()),
	}

	if len(extra) > 0 {
		fields = append(fields, extra...)
	}

	if pr, ok := peer.FromContext(ctx); ok {
		fields = append(fields, zap.String("peer.address", pr.Addr.String()))
		if s, ok := getPeerIdentity(pr); ok {
			fields = append(fields, zap.String("peer.identity", s))
		}
	}

	if grpcErr != nil {
		fields = append(fields, zap.Error(grpcErr))
	}

	var msg string
	switch l.interceptorType {
	case unaryType:
		msg = "finished unary call with code " + code.String()
	case streamType:
		msg = "finished streaming call with code " + code.String()
	}

	switch codeToLevel(code) {
	case zapcore.InfoLevel:
		l.Info(msg, fields...)
	case zapcore.WarnLevel:
		l.Warn(msg, fields...)
	default:
		l.Error(msg, fields...)
	}
}

func (l *serverLogger) LogStream(ctx context.Context, fullMethod, msg string, extra []zap.Field) {
	var pkg string
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)
	parts := strings.Split(service, ".")
	if l := len(parts); l > 1 {
		pkg = strings.Join(parts[:l-1], ".")
		service = parts[l-1]
	}

	var name, requestID, tracingID string

	// Use (reflected) request ID for logging. It _could_ be empty if it wasn't set
	// by some (external) middleware, but we stil log the legacy request ID too, so
	// it shouldn't be too big of an issue.
	requestID = panoramix.FromContext(ctx)
	if requestID == "" {
		requestID = requestid.FromContext(ctx)
	}

	if s, ok := logging.GetName(ctx); ok {
		name = s
	} else {
		name = l.name
	}
	if tp, ok := logging.GetTraceparent(ctx); ok {
		tracingID = tp.String()
		if requestID == "" {
			requestID = tp.TraceID()
		}
	}

	fields := []zap.Field{
		zap.String("name", name),
		zap.String("system", "grpc"),
		zap.String("span.kind", "server"),
		zap.String("grpc.package", pkg),
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
		zap.String("request-id", requestID),
		zap.String("tracing-id", tracingID),
	}

	if len(extra) > 0 {
		fields = append(fields, extra...)
	}

	l.Info(msg, fields...)
}

// codeToLevel returns the log level to use for a given gRPC return code.
func codeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zapcore.InfoLevel
	case codes.Canceled:
		return zapcore.InfoLevel
	case codes.Unknown:
		return zapcore.ErrorLevel
	case codes.InvalidArgument:
		return zapcore.InfoLevel
	case codes.DeadlineExceeded:
		return zapcore.WarnLevel
	case codes.NotFound:
		return zapcore.InfoLevel
	case codes.AlreadyExists:
		return zapcore.InfoLevel
	case codes.PermissionDenied:
		return zapcore.WarnLevel
	case codes.Unauthenticated:
		return zapcore.InfoLevel
	case codes.ResourceExhausted:
		return zapcore.WarnLevel
	case codes.FailedPrecondition:
		return zapcore.WarnLevel
	case codes.Aborted:
		return zapcore.WarnLevel
	case codes.OutOfRange:
		return zapcore.WarnLevel
	case codes.Unimplemented:
		return zapcore.ErrorLevel
	case codes.Internal:
		return zapcore.ErrorLevel
	case codes.Unavailable:
		return zapcore.WarnLevel
	case codes.DataLoss:
		return zapcore.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}

func getPeerIdentity(p *peer.Peer) (string, bool) {
	if p.AuthInfo == nil {
		return "", false
	}
	if p.AuthInfo.AuthType() == "tls" {
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			return getCommonName(tlsInfo.State)
		}
	}

	return "", false
}

func getCommonName(cs tls.ConnectionState) (string, bool) {
	if len(cs.PeerCertificates) == 0 || cs.PeerCertificates[0] == nil {
		return "", false
	}
	return cs.PeerCertificates[0].Subject.CommonName, true
}

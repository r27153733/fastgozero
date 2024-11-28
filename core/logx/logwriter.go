package logx

import (
	"github.com/zeromicro/go-zero/fastext"
	"log"
)

type logWriter struct {
	logger *log.Logger
}

func newLogWriter(logger *log.Logger) logWriter {
	return logWriter{
		logger: logger,
	}
}

func (lw logWriter) Close() error {
	return nil
}

func (lw logWriter) Write(data []byte) (int, error) {
	lw.logger.Print(fastext.B2s(data))
	return len(data), nil
}

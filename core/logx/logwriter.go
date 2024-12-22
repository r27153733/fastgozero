package logx

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
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
	lw.logger.Print(bytesconv.BToS(data))
	return len(data), nil
}

package log

import (
	"fmt"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type alignEncoder struct {
	zapcore.Encoder
}

// EncodeEntry is a custom method to wrap the original EncodeEntry and align the message.
func (ae *alignEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Here we can manipulate the entry.Message to align it as we want
	// For left alignment, you might want to pad it with spaces to a certain width
	entry.Message = fmt.Sprintf("%-50s", entry.Message) // Left align and pad to 50 characters

	// Call the original Encoder's EncodeEntry method with the modified entry.
	return ae.Encoder.EncodeEntry(entry, fields)
}

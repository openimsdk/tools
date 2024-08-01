// Copyright © 2024 OpenIM open source community. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

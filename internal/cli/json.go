package cli

import (
	"bytes"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MarshalJSON renders a proto message as stable, indented JSON for --json
// output. protojson deliberately randomizes its whitespace, so the output
// is re-indented through encoding/json to stay byte-stable for scripts and
// golden tests (field order is protojson's, i.e. schema order).
func MarshalJSON(m proto.Message) ([]byte, error) {
	raw, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

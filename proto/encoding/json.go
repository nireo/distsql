package encoding

import (
	"bytes"
	"encoding/json"

	store "github.com/nireo/distsql/proto"
)

// Execute result
type Result struct {
	LastInsertID int64   `json:"last_insert_id,omitempty"`
	RowsAffected int64   `json:"rows_affected,omitempty"`
	Error        string  `json:"error,omitempty"`
	Time         float64 `json:"time,omitempty"`
}

func convertExecuteResult(res *store.ExecRes) *Result {
	return &Result{
		LastInsertID: res.LastInsertId,
		RowsAffected: res.RowsAffected,
		Error:        res.Error,
		Time:         0,
	}
}

func structToJson(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	// trim out newlines such that testing is easier, and that json payloads
	// are smaller.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func ProtoToJSON(v any) ([]byte, error) {
	switch val := v.(type) {
	case *store.ExecRes:
		return structToJson(convertExecuteResult(val))
	case []*store.ExecRes:
		vals := make([]*Result, len(val))
		for i, res := range val {
			vals[i] = convertExecuteResult(res)
		}
		return structToJson(vals)
	default:
		return structToJson(val)
	}
}

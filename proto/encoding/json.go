package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"

	store "github.com/nireo/distsql/proto"
)

// Execute result
type Result struct {
	LastInsertID int64   `json:"last_insert_id,omitempty"`
	RowsAffected int64   `json:"rows_affected,omitempty"`
	Error        string  `json:"error,omitempty"`
	Time         float64 `json:"time,omitempty"`
}

type EncQueryRes struct {
	Columns []string `json:"columns,omitempty"`
	Types   []string `json:"types,omitempty"`
	Values  [][]any  `json:"values,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func convertExecuteResult(res *store.ExecRes) *Result {
	return &Result{
		LastInsertID: res.LastInsertId,
		RowsAffected: res.RowsAffected,
		Error:        res.Error,
		Time:         0,
	}
}

func convertValues(dst [][]any, v []*store.Values) error {
	for i := range v {
		if v[i] == nil {
			dst[i] = nil
			continue
		}

		params := v[i].GetParams()
		if params == nil {
			dst[i] = nil
			continue
		}

		values := make([]interface{}, len(params))
		for p := range params {
			switch pv := params[p].GetValue().(type) {
			case *store.Parameter_I:
				values[i] = pv.I
			case *store.Parameter_D:
				values[i] = pv.D
			case *store.Parameter_S:
				values[i] = pv.S
			case *store.Parameter_B:
				values[i] = pv.B
			case *store.Parameter_Y:
				values[i] = pv.Y
			case nil:
				values[i] = nil
			default:
				return fmt.Errorf("unrecognized parameter type: %T", pv)
			}
		}

		dst[i] = values
	}
	return nil
}

func convertQueryRes(q *store.QueryRes) (*EncQueryRes, error) {
	dstValues := make([][]any, len(q.Values))
	if err := convertValues(dstValues, q.Values); err != nil {
		return nil, err
	}

	return &EncQueryRes{
		Columns: q.Columns,
		Types:   q.Types,
		Values:  dstValues,
		Error:   q.Error,
	}, nil
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
	case *store.QueryRes:
		res, err := convertQueryRes(val)
		if err != nil {
			return nil, err
		}
		return structToJson(res)
	case []*store.QueryRes:
		var err error
		results := make([]*EncQueryRes, len(val))
		for i := range results {
			results[i], err = convertQueryRes(val[i])
			if err != nil {
				return nil, err
			}
		}
		return structToJson(results)
	case []*store.Values:
		vals := make([][]any, len(val))
		if err := convertValues(vals, val); err != nil {
			return nil, err
		}
		return structToJson(vals)
	default:
		return structToJson(val)
	}
}

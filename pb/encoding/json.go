package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nireo/distsql/pb"
)

// Execute result
type Result struct {
	LastInsertID int64   `json:"last_insert_id,omitempty"`
	RowsAffected int64   `json:"rows_affected,omitempty"`
	Error        string  `json:"error,omitempty"`
	Time         float64 `json:"time,omitempty"`
}

type EncQueryRes struct {
	Columns []string        `json:"columns,omitempty"`
	Types   []string        `json:"types,omitempty"`
	Values  [][]interface{} `json:"values,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func convertExecuteResult(res *pb.ExecRes) *Result {
	return &Result{
		LastInsertID: res.LastInsertId,
		RowsAffected: res.RowsAffected,
		Error:        res.Error,
		Time:         0,
	}
}

func convertValues(dest [][]interface{}, v []*pb.Values) error {
	for n := range v {
		vals := v[n]
		if vals == nil {
			dest[n] = nil
			continue
		}

		params := vals.GetParams()
		if params == nil {
			dest[n] = nil
			continue
		}

		rowValues := make([]interface{}, len(params))
		for p := range params {
			switch w := params[p].GetValue().(type) {
			case *pb.Parameter_I:
				rowValues[p] = w.I
			case *pb.Parameter_D:
				rowValues[p] = w.D
			case *pb.Parameter_B:
				rowValues[p] = w.B
			case *pb.Parameter_Y:
				rowValues[p] = w.Y
			case *pb.Parameter_S:
				rowValues[p] = w.S
			case nil:
				rowValues[p] = nil
			default:
				return fmt.Errorf("unsupported parameter type at index %d: %T", p, w)
			}
		}
		dest[n] = rowValues
	}

	return nil
}

func convertQueryRes(q *pb.QueryRes) (*EncQueryRes, error) {
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
	case *pb.ExecRes:
		return structToJson(convertExecuteResult(val))
	case []*pb.ExecRes:
		vals := make([]*Result, len(val))
		for i, res := range val {
			vals[i] = convertExecuteResult(res)
		}
		return structToJson(vals)
	case *pb.QueryRes:
		res, err := convertQueryRes(val)
		if err != nil {
			return nil, err
		}
		return structToJson(res)
	case []*pb.QueryRes:
		var err error
		results := make([]*EncQueryRes, len(val))
		for i := range results {
			results[i], err = convertQueryRes(val[i])
			if err != nil {
				return nil, err
			}
		}
		return structToJson(results)
	case []*pb.Values:
		vals := make([][]any, len(val))
		if err := convertValues(vals, val); err != nil {
			return nil, err
		}
		return structToJson(vals)
	default:
		return structToJson(val)
	}
}

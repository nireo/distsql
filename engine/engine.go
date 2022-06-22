package engine

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	store "github.com/nireo/distsql/proto"
)

type Engine struct {
	readDB   *sql.DB
	writeDB  *sql.DB
	readOnly bool
	isMem    bool
	path     string
}

// Open creates a new database engine instance. It starts a read-only and a write connection.
func Open(path string) (*Engine, error) {
	// write and read database connection
	writeDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, err
	}

	log.Printf("created write connection")

	// read-only database connection
	readDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return nil, err
	}

	log.Printf("created read connection")

	if err := writeDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed database ping: %s", err)
	}

	log.Printf("pinged database connection")

	return &Engine{
		readDB:   readDB,
		writeDB:  writeDB,
		path:     path,
		isMem:    false,
		readOnly: false,
	}, nil
}

func (eng *Engine) Path() string {
	return eng.path
}

func (eng *Engine) IsMem() bool {
	return eng.isMem
}

func (eng *Engine) IsReadOnly() bool {
	return eng.readOnly
}

// Close the database connections
func (eng *Engine) Close() error {
	if err := eng.readDB.Close(); err != nil {
		return err
	}

	return eng.writeDB.Close()
}

func (eng *Engine) ExecString(str string) ([]*store.ExecRes, error) {
	req := &store.Request{
		Statements: []*store.Statement{
			{Sql: str},
		},
	}

	return eng.Exec(req)
}

func (eng *Engine) Exec(req *store.Request) ([]*store.ExecRes, error) {
	conn, err := eng.writeDB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// we are only using the connection at this point
	// TODO: implement the transaction version

	results := make([]*store.ExecRes, 0)

	for _, stmt := range req.Statements {
		res := &store.ExecRes{}

		if stmt.Sql == "" {
			continue
		}

		sqlParams, err := convertParamsToSQL(stmt.Params)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		sqlRes, err := conn.ExecContext(context.Background(), stmt.Sql, sqlParams...)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		res.RowsAffected, err = sqlRes.RowsAffected()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		res.LastInsertId, err = sqlRes.LastInsertId()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		results = append(results, res)
	}

	return results, nil
}

func (eng *Engine) QueryString(query string) ([]*store.QueryRes, error) {
	r := &store.Request{
		Statements: []*store.Statement{
			{Sql: query},
		},
	}
	return eng.Query(r)
}

func (eng *Engine) Query(req *store.Request) ([]*store.QueryRes, error) {
	// get database connection from the read database
	conn, err := eng.readDB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// TODO: Implement transaction
	results := make([]*store.QueryRes, 0)

	for _, q := range req.Statements {
		if q.Sql == "" {
			continue
		}

		res := &store.QueryRes{}
		var readOnly bool

		// This code is used to make sure that the given query, does
		// not actually try to modify the database.
		if err := conn.Raw(func(dConn any) error {
			c := dConn.(*sqlite3.SQLiteConn)
			stmt, err := c.Prepare(q.Sql)
			if err != nil {
				return err
			}
			defer stmt.Close()

			sstmt := stmt.(*sqlite3.SQLiteStmt)
			readOnly = sstmt.Readonly()
			return nil
		}); err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		// tried to modify the database
		if !readOnly {
			res.Error = "tried to change the database in a query operation"
			results = append(results, res)
			continue
		}

		params, err := convertParamsToSQL(q.Params)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		r, err := conn.QueryContext(context.Background(), q.Sql, params...)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}
		defer r.Close()

		types, err := r.ColumnTypes()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}
		formattedTypes := make([]string, len(types))
		for i := range types {
			formattedTypes[i] = strings.ToLower(types[i].DatabaseTypeName())
		}

		cols, err := r.Columns()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		for r.Next() { // scan rows
			dest := make([]any, len(cols))
			destPtrs := make([]any, len(dest))
			for i := range dest {
				destPtrs[i] = &dest[i]
			}

			// Scan copies the the columns in the current row into the values pointed at by dest
			if err := r.Scan(destPtrs...); err != nil {
				return nil, err
			}

			prs, err := convertToProto(formattedTypes, dest)
			if err != nil {
				return nil, err
			}

			res.Values = append(res.Values, &store.Values{
				Params: prs,
			})
		}

		if err := r.Err(); err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		// we set the values here, since if something fails, we don't want
		// to return a json response where half of the items are missing
		// and half are there.
		res.Columns = cols
		res.Types = formattedTypes
		results = append(results, res)
	}

	return results, nil
}

func convertParamsToSQL(params []*store.Parameter) ([]any, error) {
	if params == nil {
		return nil, nil
	}

	values := make([]any, 0)
	for _, p := range params {
		switch p.GetValue().(type) {
		case *store.Parameter_I:
			values = append(values, sql.Named(p.GetName(), p.GetI()))
		case *store.Parameter_D:
			values = append(values, sql.Named(p.GetName(), p.GetD()))
		case *store.Parameter_B:
			values = append(values, sql.Named(p.GetName(), p.GetB()))
		case *store.Parameter_Y:
			values = append(values, sql.Named(p.GetName(), p.GetY()))
		case *store.Parameter_S:
			values = append(values, sql.Named(p.GetName(), p.GetS()))
		default:
			return nil, fmt.Errorf("unsupported parameter type: %T", params)
		}
	}

	return values, nil
}

func convertToProto(types []string, row []any) ([]*store.Parameter, error) {
	values := make([]*store.Parameter, 0)

	for i, val := range row {
		switch v := val.(type) {
		case int:
		case int64:
			values = append(values, &store.Parameter{
				Value: &store.Parameter_I{I: v},
			})
		case float64:
			values = append(values, &store.Parameter{
				Value: &store.Parameter_D{D: v},
			})
		case bool:
			values = append(values, &store.Parameter{
				Value: &store.Parameter_B{B: v},
			})
		case string:
			values = append(values, &store.Parameter{
				Value: &store.Parameter_S{S: v},
			})
		case []byte:
			if isStringType(types[i]) {
				values = append(values, &store.Parameter{
					Value: &store.Parameter_S{S: string(v)},
				})
			} else {
				values = append(values, &store.Parameter{
					Value: &store.Parameter_Y{Y: v},
				})
			}
		case time.Time:
			str, err := v.MarshalText()
			if err != nil {
				return nil, err
			}
			values[i] = &store.Parameter{
				Value: &store.Parameter_S{
					S: string(str),
				},
			}
		case nil:
			continue
		default:
			return nil, fmt.Errorf("unrecognized type: %T", val)
		}
	}

	return values, nil
}

func isStringType(t string) bool {
	return t == "text" ||
		t == "json" ||
		t == "" ||
		strings.HasPrefix(t, "varchar") ||
		strings.HasPrefix(t, "varying character") ||
		strings.HasPrefix(t, "nchar") ||
		strings.HasPrefix(t, "native character") ||
		strings.HasPrefix(t, "nvarchar") ||
		strings.HasPrefix(t, "clob")
}

func (eng *Engine) FileSize() (int64, error) {
	if eng.isMem {
		return 0, nil
	}
	fi, err := os.Stat(eng.path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

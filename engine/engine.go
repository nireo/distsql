package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/nireo/distsql/pb"
)

// Engine handles database operations such that queries and executes.
// This also handles backups and copying the data. Also collects metrics
// for node information.
type Engine struct {
	readDB   *sql.DB
	writeDB  *sql.DB
	readOnly bool
	isMem    bool
	path     string
}

// Open creates a new database engine instance.
// It starts a read-only and a write connection.
func Open(path string) (*Engine, error) {
	// write and read database connection
	writeDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, err
	}

	// read-only database connection
	readDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return nil, err
	}

	if err := writeDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed database ping: %s", err)
	}

	return &Engine{
		readDB:   readDB,
		writeDB:  writeDB,
		path:     path,
		isMem:    false,
		readOnly: false,
	}, nil
}

func OpenMemory() (*Engine, error) {
	memPath := fmt.Sprintf("file:/%s", genRandomString())

	rwOpts := []string{
		"mode=rw",
		"vfs=memdb",
		"_txlock=immediate",
	}

	rwDB, err := sql.Open(
		"sqlite3",
		fmt.Sprintf("%s?%s", memPath, strings.Join(rwOpts, "&")),
	)
	if err != nil {
		return nil, err
	}

	roOpts := []string{
		"mode=ro",
		"vfs=memdb",
		"_txlock=deferred",
	}

	roDB, err := sql.Open(
		"sqlite3",
		fmt.Sprintf("%s?%s", memPath, strings.Join(roOpts, "&")),
	)
	if err != nil {
		return nil, err
	}

	if err := rwDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping in-memory database: %s", err.Error())
	}

	return &Engine{
		isMem:   true,
		path:    ":memory:",
		writeDB: rwDB,
		readDB:  roDB,
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

func (eng *Engine) ExecString(str string) ([]*pb.ExecRes, error) {
	req := &pb.Request{
		Statements: []*pb.Statement{
			{Sql: str},
		},
	}

	return eng.Exec(req)
}

func (eng *Engine) Exec(req *pb.Request) ([]*pb.ExecRes, error) {
	conn, err := eng.writeDB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	type Runner interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	}

	var runner Runner
	var transaction *sql.Tx
	if req.Transaction {
		transaction, err = conn.BeginTx(context.Background(), nil)
		if err != nil {
			return nil, err
		}

		defer func() {
			if transaction != nil {
				transaction.Rollback()
			}
		}()

		runner = transaction
	} else {
		runner = conn
	}

	results := make([]*pb.ExecRes, 0)

	// returns bool determining if the loop should break
	errorHandler := func(res *pb.ExecRes, err error) bool {
		res.Error = err.Error()
		results = append(results, res)

		if transaction != nil {
			transaction.Rollback()
			transaction = nil
			return false
		}

		return true
	}

	for _, stmt := range req.Statements {
		res := &pb.ExecRes{}

		if stmt.Sql == "" {
			continue
		}

		sqlParams, err := convertParamsToSQL(stmt.Params)
		if err != nil {
			if errorHandler(res, err) {
				continue
			}
			break
		}

		sqlRes, err := runner.ExecContext(context.Background(), stmt.Sql, sqlParams...)
		if err != nil {
			if errorHandler(res, err) {
				continue
			}
			break
		}

		res.RowsAffected, err = sqlRes.RowsAffected()
		if err != nil {
			if errorHandler(res, err) {
				continue
			}
			break
		}

		res.LastInsertId, err = sqlRes.LastInsertId()
		if err != nil {
			if errorHandler(res, err) {
				continue
			}
			break
		}

		results = append(results, res)
	}

	if transaction != nil {
		err = transaction.Commit()
	}

	return results, err
}

func (eng *Engine) QueryString(query string) ([]*pb.QueryRes, error) {
	r := &pb.Request{
		Statements: []*pb.Statement{
			{Sql: query},
		},
	}
	return eng.Query(r)
}

func (eng *Engine) Query(req *pb.Request) ([]*pb.QueryRes, error) {
	var err error
	// get connection from the read database
	conn, err := eng.readDB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	type Runner interface {
		QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	}
	var runner Runner
	var transaction *sql.Tx
	if req.Transaction {
		transaction, err = conn.BeginTx(context.Background(), nil)
		if err != nil {
			return nil, err
		}
		defer transaction.Rollback()
		runner = transaction
	} else {
		runner = conn
	}

	results := make([]*pb.QueryRes, 0)
	for _, q := range req.Statements {
		if q.Sql == "" {
			continue
		}

		res := &pb.QueryRes{}
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

		r, err := runner.QueryContext(context.Background(), q.Sql, params...)
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
			for i := range destPtrs {
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

			res.Values = append(res.Values, &pb.Values{
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

	if transaction != nil {
		err = transaction.Commit()
	}

	return results, err
}

func convertParamsToSQL(params []*pb.Parameter) ([]any, error) {
	values := make([]interface{}, len(params))
	for i := range params {
		switch w := params[i].GetValue().(type) {
		case *pb.Parameter_I:
			values[i] = sql.Named(params[i].GetName(), w.I)
		case *pb.Parameter_D:
			values[i] = sql.Named(params[i].GetName(), w.D)
		case *pb.Parameter_B:
			values[i] = sql.Named(params[i].GetName(), w.B)
		case *pb.Parameter_Y:
			values[i] = sql.Named(params[i].GetName(), w.Y)
		case *pb.Parameter_S:
			values[i] = sql.Named(params[i].GetName(), w.S)
		default:
			return nil, fmt.Errorf("unsupported type: %T", w)
		}
	}
	return values, nil
}

func convertToProto(types []string, row []any) ([]*pb.Parameter, error) {
	values := make([]*pb.Parameter, len(types))
	for i, v := range row {
		switch val := v.(type) {
		case int:
		case int64:
			values[i] = &pb.Parameter{
				Value: &pb.Parameter_I{
					I: val,
				},
			}
		case float64:
			values[i] = &pb.Parameter{
				Value: &pb.Parameter_D{
					D: val,
				},
			}
		case bool:
			values[i] = &pb.Parameter{
				Value: &pb.Parameter_B{
					B: val,
				},
			}
		case string:
			values[i] = &pb.Parameter{
				Value: &pb.Parameter_S{
					S: val,
				},
			}
		case []byte:
			if isStringType(types[i]) {
				values[i].Value = &pb.Parameter_S{
					S: string(val),
				}
			} else {
				values[i] = &pb.Parameter{
					Value: &pb.Parameter_Y{
						Y: val,
					},
				}
			}
		case time.Time:
			rfc3339, err := val.MarshalText()
			if err != nil {
				return nil, err
			}
			values[i] = &pb.Parameter{
				Value: &pb.Parameter_S{
					S: string(rfc3339),
				},
			}
		case nil:
			continue
		default:
			return nil, fmt.Errorf("unhandled column type: %T %v", val, val)
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

// FileSize size of the sqlite file on disk.
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

// Size returns the database size in bytes.
func (eng *Engine) Size() (int64, error) {
	rows, err := eng.QueryString(
		`SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()`,
	)
	if err != nil {
		return 0, err
	}

	return rows[0].Values[0].Params[0].GetI(), nil
}

// Serialize returns the database as bytes.
func (eng *Engine) Serialize() ([]byte, error) {
	if eng.isMem {
		return nil, errors.New("in-memory database cannot be serialized")
	}
	return os.ReadFile(eng.path)
}

// Backup copies the data from connections to a given path.
func (eng *Engine) Backup(path string) error {
	dstDB, err := Open(path)
	if err != nil {
		return err
	}

	if err := copyEngine(dstDB, eng); err != nil {
		return fmt.Errorf("copying database failed")
	}
	return nil
}

func copyConnection(dst, src *sqlite3.SQLiteConn) error {
	backup, err := dst.Backup("main", src, "main")
	if err != nil {
		return err
	}

	for {
		done, err := backup.Step(-1)
		if err != nil {
			if err := backup.Finish(); err != nil {
				return err
			}
			return err
		}

		if done {
			break
		}

		time.Sleep(250 * time.Millisecond)
	}
	return backup.Finish()
}

func copyEngine(dst, src *Engine) error {
	dstConn, err := dst.writeDB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer dstConn.Close()

	srcConn, err := src.writeDB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer srcConn.Close()

	var dstSQConn *sqlite3.SQLiteConn
	backup := func(driverConn interface{}) error {
		srcSqlite := driverConn.(*sqlite3.SQLiteConn)
		return copyConnection(dstSQConn, srcSqlite)
	}

	return dstConn.Raw(
		func(driverConn interface{}) error {
			dstSQConn = driverConn.(*sqlite3.SQLiteConn)
			return srcConn.Raw(backup)
		})
}

// Metric gets some data from the database. Basically information about
// how much data has been used etc...
func (eng *Engine) Metric() (map[string]any, error) {
	memData, err := (func() (map[string]int64, error) {
		ms := make(map[string]int64)
		for _, p := range []string{
			"max_page_count",
			"page_count",
			"page_size",
			"hard_heap_limit",
			"soft_heap_limit",
			"cache_size",
			"freelist_count",
		} {
			res, err := eng.QueryString(fmt.Sprintf("PRAGMA %s", p))
			if err != nil {
				return nil, err
			}
			ms[p] = res[0].Values[0].Params[0].GetI()
		}

		return ms, nil
	})()
	if err != nil {
		return nil, err
	}

	databaseSize, err := eng.Size()
	if err != nil {
		return nil, err
	}

	resultMap := map[string]any{
		"memory_stats": memData,
		"size":         databaseSize,
	}
	return resultMap, nil
}

func (eng *Engine) Copy(dst *Engine) error {
	return copyEngine(dst, eng)
}

// genRandomString is used for random temporary connection names.
func genRandomString() string {
	var output strings.Builder
	chars := "abcdedfghijklmnopqrstABCDEFGHIJKLMNOP"

	for i := 0; i < 20; i++ {
		random := rand.Intn(len(chars))
		randomChar := chars[random]
		output.WriteString(string(randomChar))
	}
	return output.String()
}

package engine_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/nireo/distsql/engine"
	"github.com/nireo/distsql/pb"
	"github.com/nireo/distsql/pb/encoding"
	"github.com/stretchr/testify/require"
)

func createTestEngine(t *testing.T) (*engine.Engine, error) {
	t.Helper()
	dir, err := ioutil.TempDir("", "distsql-test-")
	if err != nil {
		t.Fatal(err)
	}

	dbpath := path.Join(dir, "test.db")

	eng, err := engine.Open(dbpath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
		eng.Close()
	})

	return eng, nil
}

func TestCreation(t *testing.T) {
	eng, err := createTestEngine(t)
	if err != nil {
		t.Fatal(err)
	}

	if eng.IsMem() {
		t.Fatalf("is memory database even though it shouldn't be")
	}

	if eng.IsReadOnly() {
		t.Fatalf("is read-only even though it shouldn't be")
	}

	if err = eng.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateTable(t *testing.T) {
	eng, err := createTestEngine(t)
	if err != nil {
		t.Fatal(err)
	}

	res, err := eng.ExecString("CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %s", err.Error())
	}

	jsonRes := convertToJSON(res)
	if jsonRes != "[{}]" {
		t.Fatalf("unrecognized output. want=%s got=%s", "[{}]", jsonRes)
	}

	query, err := eng.QueryString("SELECT * FROM test")
	if err != nil {
		t.Fatalf("failed reading table: %s", err.Error())
	}

	qjson := convertToJSON(query)
	if qjson != `[{"columns":["id","name"],"types":["integer","text"]}]` {
		t.Fatalf("results don't match: %s", qjson)
	}
}

func TestDoesntExist(t *testing.T) {
	eng, err := createTestEngine(t)
	if err != nil {
		t.Fatal(err)
	}

	query, err := eng.QueryString("SELECT * FROM test")
	if err != nil {
		t.Fatalf("failed to query table: %s", err.Error())
	}

	qjson := convertToJSON(query)
	if qjson != `[{"error":"no such table: test"}]` {
		t.Fatalf("results don't match: %s", qjson)
	}
}

func TestGetSizes(t *testing.T) {
	eng, err := createTestEngine(t)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := eng.Size(); err != nil {
		t.Fatalf("eng.Size() failed: %s", err)
	}

	if _, err := eng.FileSize(); err != nil {
		t.Fatalf("eng.FileSize() failed: %s", err)
	}
}

func TestEmptyStatement(t *testing.T) {
	eng, err := createTestEngine(t)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := eng.ExecString(""); err != nil {
		t.Fatalf("failed to execute: %s", err.Error())
	}

	if _, err := eng.ExecString(";"); err != nil {
		t.Fatalf("failed to execute: %s", err.Error())
	}
}

func TestSimpleUsage(t *testing.T) {
	db, err := createTestEngine(t)
	if err != nil {
		t.Fatalf("failed to create engine")
	}

	_, err = db.ExecString("CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %s", err.Error())
	}

	_, err = db.ExecString(`INSERT INTO test(name) VALUES("atest")`)
	if err != nil {
		t.Fatalf("failed to insert record: %s", err.Error())
	}

	_, err = db.ExecString(`INSERT INTO test(name) VALUES("btest")`)
	if err != nil {
		t.Fatalf("failed to insert record: %s", err.Error())
	}

	r, err := db.QueryString(`SELECT * FROM test`)
	if err != nil {
		t.Fatalf("failed to query table: %s", err.Error())
	}

	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"atest"],[2,"btest"]]}]`, convertToJSON(r); exp != got {
		t.Fatalf("unexpected results for query\nexp: %s\ngot: %s", exp, got)
	}

	r, err = db.QueryString(`SELECT * FROM test WHERE name="btest"`)
	if err != nil {
		t.Fatalf("failed to query table: %s", err.Error())
	}

	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[2,"btest"]]}]`, convertToJSON(r); exp != got {
		t.Fatalf("unexpected results for query\nexp: %s\ngot: %s", exp, got)
	}
}

func convertToJSON(a any) string {
	j, err := encoding.ProtoToJSON(a)
	if err != nil {
		return ""
	}

	return string(j)
}

func TestCopy(t *testing.T) {
	src, err := createTestEngine(t)
	if err != nil {
		t.Fatalf("failed to create engine")
	}

	_, err = src.ExecString("CREATE TABLE foo(id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	req := &pb.Request{
		Transaction: true,
		Statements: []*pb.Statement{
			{
				Sql: `INSERT INTO foo(id, name) VALUES(1, "test")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(2, "test")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(3, "test")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(4, "test")`,
			},
		},
	}

	_, err = src.Exec(req)
	require.NoError(t, err)

	dst, err := createTestEngine(t)
	require.NoError(t, err)

	err = src.Copy(dst)
	require.NoError(t, err)

	res, err := dst.QueryString(`SELECT * FROM foo`)

	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"test"],[2,"test"],[3,"test"],[4,"test"]]}]`, convertToJSON(res); exp != got {
		t.Fatalf("unexpected results for query\nexp: %s\ngot: %s", exp, got)
	}
}

func TestSerialize(t *testing.T) {
	db, err := createTestEngine(t)
	require.NoError(t, err)

	_, err = db.ExecString("CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %s", err.Error())
	}

	req := &pb.Request{
		Transaction: true,
		Statements: []*pb.Statement{
			{
				Sql: `INSERT INTO foo(id, name) VALUES(1, "fiona")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(2, "fiona")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(3, "fiona")`,
			},
			{
				Sql: `INSERT INTO foo(id, name) VALUES(4, "fiona")`,
			},
		},
	}
	_, err = db.Exec(req)
	require.NoError(t, err)

	dstDB, err := ioutil.TempFile("", "serialize-test-")
	require.NoError(t, err)
	dstDB.Close()
	defer os.Remove(dstDB.Name())

	b, err := db.Serialize()
	require.NoError(t, err)
	err = ioutil.WriteFile(dstDB.Name(), b, 0644)
	require.NoError(t, err)

	newDB, err := engine.Open(dstDB.Name())
	require.NoError(t, err)
	defer newDB.Close()
	defer os.Remove(dstDB.Name())

	ro, err := newDB.QueryString(`SELECT * FROM foo`)
	require.NoError(t, err)

	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"fiona"],[2,"fiona"],[3,"fiona"],[4,"fiona"]]}]`, convertToJSON(ro); exp != got {
		t.Fatalf("unexpected results for query\nexp: %s\ngot: %s", exp, got)
	}
}

package engine_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/nireo/distsql/engine"
	"github.com/nireo/distsql/proto/encoding"
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

	_, err = db.ExecString("CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %s", err.Error())
	}

	_, err = db.ExecString(`INSERT INTO foo(name) VALUES("fiona")`)
	if err != nil {
		t.Fatalf("failed to insert record: %s", err.Error())
	}

	_, err = db.ExecString(`INSERT INTO foo(name) VALUES("aoife")`)
	if err != nil {
		t.Fatalf("failed to insert record: %s", err.Error())
	}

	r, err := db.QueryString(`SELECT * FROM foo`)
	if err != nil {
		t.Fatalf("failed to query table: %s", err.Error())
	}
	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"fiona"],[2,"aoife"]]}]`, convertToJSON(r); exp != got {
		t.Fatalf("unexpected results for query\nexp: %s\ngot: %s", exp, got)
	}

	r, err = db.QueryString(`SELECT * FROM foo WHERE name="aoife"`)
	if err != nil {
		t.Fatalf("failed to query table: %s", err.Error())
	}
	if exp, got := `[{"columns":["id","name"],"types":["integer","text"],"values":[[2,"aoife"]]}]`, convertToJSON(r); exp != got {
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

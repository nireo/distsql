package engine_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/nireo/distsql/engine"
)

func TestCreation(t *testing.T) {
	dir, err := ioutil.TempDir("", "distsql-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbpath := path.Join(dir, "testdb")

	eng, err := engine.Open(dbpath)
	if err != nil {
		t.Fatal(err)
	}

	if eng == nil {
		t.Fatalf("engine instance nil")
	}

	if eng.Path() != dbpath {
		t.Fatalf("engine path and db path don't match. want: %s | got %s", dbpath, eng.Path())
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

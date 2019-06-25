package test

import (
	"github.com/Yimismi/sql2go"
	"os"
	"testing"
)

func TestFromFile1(t *testing.T) {
	ares := sql2go.NewConvertArgs().SetGenJson(true).SetPackageName("test")
	code, err := sql2go.FromFile("./1.sql", ares)
	if err != nil {
		t.Error(err)
		return
	}
	f, err := os.Create("db_struct1.go")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	f.Write(code)
}
func TestFromFile2(t *testing.T) {
	ares := sql2go.NewConvertArgs().SetGenJson(true).
		SetPackageName("test").SetTmpl(sql2go.GOTMPL).
		SetColPrefix("f_").
		SetTablePrefix("t_")

	code, err := sql2go.FromFile("./2.sql", ares)
	if err != nil {
		t.Error(err)
		return
	}
	f, err := os.Create("db_struct2.go")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	f.Write(code)
}

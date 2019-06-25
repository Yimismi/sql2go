package test

import (
	"github.com/Yimismi/sql2go"
	"os"
	"testing"
)

func TestFromFile1(t *testing.T) {
	args := sql2go.NewConvertArgs().SetGenJson(true).SetPackageName("test")
	code, err := sql2go.FromFile("./1.sql", args)
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
	args := sql2go.NewConvertArgs().
		SetPackageName("test").SetTmpl(sql2go.GOTMPL).
		SetColPrefix("f_").
		SetTablePrefix("t_")

	code, err := sql2go.FromFile("./2.sql", args)
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

func TestFromSql1(t *testing.T) {
	sql := `
CREATE TABLE IF NOT EXISTS t_person (
  f_age INT(11) NULL,
  f_id INT(11) PRIMARY KEY AUTO_INCREMENT NOT NULL,
  f_name VARCHAR(30) NOT NULL,
  f_sex VARCHAR(2) NULL
  ) ENGINE=InnoDB;
`
	args := sql2go.NewConvertArgs().SetGenJson(true).
		SetPackageName("test").
		SetColPrefix("f_").
		SetTablePrefix("t_")

	code, err := sql2go.FromSql(sql, args)
	if err != nil {
		t.Error(err)
		return
	}
	f, err := os.Create("db_struct3.go")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	f.Write(code)
}

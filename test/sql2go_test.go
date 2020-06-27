package test

import (
	"github.com/Yimismi/sql2go"
	"testing"
)

func TestFromFile1(t *testing.T) {
	args := sql2go.NewConvertArgs().SetGenJson(true)
	code, err := sql2go.FromFile("./1.sql", args)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(code))
}
func TestFromFile2(t *testing.T) {
	args := sql2go.NewConvertArgs().
		SetGenXorm(true).
		SetColPrefix("f_").
		SetTablePrefix("t_")

	code, err := sql2go.FromFile("./2.sql", args)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(code))
}

func TestFromSql1(t *testing.T) {
	sql := `
CREATE TABLE IF NOT EXISTS t_person (
  f_age INT(11) NULL,
  f_id INT(11) PRIMARY KEY AUTO_INCREMENT NOT NULL,
  f_name VARCHAR(30) NOT NULL,
  f_sex VARCHAR(2) NULL,
  f_test TEXT
  ) ENGINE=InnoDB;
`
	args := sql2go.NewConvertArgs().
		SetGenJson(true).
		SetGenXorm(true).
		SetColPrefix("f_").
		SetTablePrefix("t_").
		SetOtherTags("db,json xlsx")

	code, err := sql2go.FromSql(sql, args)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(code))
}

package spellsql

import "testing"

type User struct {
	Name string `json:"name,omitempty"`
	Age  int    `json:"age,omitempty"`
	Addr string `json:"addr,omitempty"`
}

func TestInsert(t *testing.T) {
	u := User{
		Name: "xue",
		Age:  10,
		Addr: "南部",
	}
	s := NewSession(nil)
	s.InsertForObj(&u)
}

package data

import (
	"reflect"
	"testing"
)

func TestInvFiles(t *testing.T) {
	db := &DB{
		Items:    []*ItemBase{{InvFile: "invHax", UniqueInvFile: "invGnash"}, {InvFile: "invhax", SetInvFile: "invset"}, {}},
		Uniques:  []*UniqueItem{{InvFile: "invsoj"}, {}},
		SetItems: []*SetItem{{InvFile: ""}, {InvFile: "invtw"}},
	}
	want := []string{"invgnash", "invhax", "invset", "invsoj", "invtw"}
	if got := db.InvFiles(); !reflect.DeepEqual(got, want) {
		t.Errorf("%v want %v", got, want)
	}
	if (*DB)(nil).InvFiles() != nil {
		t.Error("nil db")
	}
}

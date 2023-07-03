package bencode

import (
	"log"
	"os"
	"reflect"
	"testing"
)

func assertAreEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("got: %v  want: %v", got, want)
	}
}
func assertAreEqualDeep(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v want: %v", got, want)
	}
}

func TestNumberParse(T *testing.T) {

	T.Run("Positive number", func(t *testing.T) {
		input := []byte("i322ed:5:")
		gotValue, gotNextIdx := numberParse(0, &input)
		wantValue, wantNextIdx := 322, 5

		assertAreEqual(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)

	})
	T.Run("Negative number", func(t *testing.T) {
		input := []byte("i-322ed:5:")
		gotValue, gotNextIdx := numberParse(0, &input)
		wantValue, wantNextIdx := -322, 6

		assertAreEqual(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)
	})
}

func TestStringParse(T *testing.T) {

	T.Run("String test 1", func(t *testing.T) {
		input := []byte("5:color4:blue")
		gotValue, gotNextIdx := stringParse(0, &input)
		wantValue, wantNextIdx := "color", 7

		assertAreEqual(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)

	})
	T.Run("String test 2", func(t *testing.T) {
		input := []byte("15:metallica_rocksd:4:color")
		gotValue, gotNextIdx := stringParse(0, &input)
		wantValue, wantNextIdx := "metallica_rocks", 18

		assertAreEqual(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)
	})
}

func TestListParse(T *testing.T) {
	T.Run("list of strings", func(t *testing.T) {
		input := []byte("l4:spam4:eggsed:5color")
		gotValue, gotNextIdx := listParse(0, &input)
		var wantValue []interface{}
		wantValue = append(wantValue, "spam", "eggs")
		wantNextIdx := 14
		assertAreEqualDeep(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)
	})
	T.Run("list of numbers", func(t *testing.T) {
		input := []byte("li322ei400eed:5color")
		gotValue, gotNextIdx := listParse(0, &input)
		var wantValue []interface{}
		wantValue = append(wantValue, 322, 400)
		wantNextIdx := 12
		assertAreEqualDeep(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)
	})
}

func TestMapParse(T *testing.T) {
	T.Run("map with string and list inside", func(t *testing.T) {
		input := []byte("d13:favorite_band4:tool6:othersl5:qotsaee5:color")
		gotValue, gotNextIdx := mapParse(0, &input)
		wantValue := make(map[string]interface{})
		wantValue["favorite_band"] = "tool"
		wantValue["others"] = []interface{}{"qotsa"}
		wantValue["byte_offsets"] = []int{0, 41}
		wantNextIdx := 41
		assertAreEqualDeep(t, gotValue, wantValue)
		assertAreEqual(t, gotNextIdx, wantNextIdx)
	})
}

func TestDecode(T *testing.T) {

	files, err := os.ReadDir("./torrent_files_test")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		T.Run(f.Name(), func(t *testing.T) {
			data, _ := os.ReadFile("./torrent_files_test/" + f.Name())
			result, _ := Decode(data)
			t.Log(result["info"].(map[string]interface{})["name"])
		})
	}

}

package data

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestJsMarshal(t *testing.T) {
	//type tmpStruct struct {
	//	A string `socket:"a"`
	//	B string `socket:"b"`
	//	C *struct {
	//		A int  `socket:"a"`
	//		B bool `socket:"b"`
	//	} `socket:"c"`
	//}
	//tmpData := &tmpStruct{}
	tmpData := make(map[string]interface{})
	if err := UnmarshalByFilePath("js_marshal.txt", &tmpData); err != nil {
		panic(err)
	}

	type tmpStruct struct {
		Id   int    `socket:"id"`
		Name string `socket:"name"`
		Org  *struct {
			Name string `socket:"name"`
			Code string `socket:"code"`
		} `socket:"org"`
	}

	tmpArrData := make([]*tmpStruct, 0)
	if err := UnmarshalByFilePath("js_arr_marshal.txt", &tmpArrData); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", tmpData)
}

func TestMarshal(t *testing.T) {
	type tmpStruct struct {
		A string `socket:"a"`
		B *struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		} `socket:"b"`
		D bool `socket:"d"`
		E int  `socket:"e"`
	}
	tmpData := &tmpStruct{
		A: "hahah",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 10,
	}
	p, err := Marshal(tmpData)
	if err != nil {
		panic(err)
	}
	readTmpData := &tmpStruct{}
	if err = UnmarshalByFilePath(p, &readTmpData); err != nil {
		panic(err)
	}
	os.RemoveAll(filepath.Dir(p))

	fmt.Println(p)
}

func TestMarshalTopArray(t *testing.T) {
	type tmpStruct struct {
		A string `socket:"a"`
		B *struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		} `socket:"b"`
		D bool `socket:"d"`
		E int  `socket:"e"`
	}
	tmpData := make([]*tmpStruct, 0, 8)
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah1",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 11,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah2",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 12,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah3",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 13,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah4",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 14,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah5",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 15,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah6",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 16,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah7",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 17,
	})
	tmpData = append(tmpData, &tmpStruct{
		A: "hahah8",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 18,
	})
	p, err := Marshal(tmpData)
	if err != nil {
		panic(err)
	}
	readTmpData := make([]*tmpStruct, 0, 8)
	if err = UnmarshalByFilePath(p, &readTmpData); err != nil {
		panic(err)
	}
	os.RemoveAll(filepath.Dir(p))

	fmt.Println(p)
}

func TestMarshalStrArr(t *testing.T) {
	tmpData := []string{"hello", "world", "!!"}
	p, err := Marshal(tmpData)
	if err != nil {
		panic(err)
	}

	readTmpData := make([]string, 0)
	if err = UnmarshalByFilePath(p, &readTmpData); err != nil {
		panic(err)
	}

	fmt.Println(readTmpData)
}

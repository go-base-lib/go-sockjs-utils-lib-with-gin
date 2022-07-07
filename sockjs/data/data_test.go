package data

import (
	"fmt"
	"io/ioutil"
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
		A: "哈哈1",
		B: &struct {
			C []string `socket:"c"`
			D string   `socket:"d"`
		}{C: []string{"hello", "world"}, D: "asd"},
		D: true,
		E: 11,
	})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah2",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 12,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah3",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 13,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah4",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 14,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah5",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 15,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah6",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 16,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah7",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 17,
	//})
	//tmpData = append(tmpData, &tmpStruct{
	//	A: "hahah8",
	//	B: &struct {
	//		C []string `socket:"c"`
	//		D string   `socket:"d"`
	//	}{C: []string{"hello", "world"}, D: "asd"},
	//	D: true,
	//	E: 18,
	//})
	p, err := Marshal(tmpData)
	if err != nil {
		panic(err)
	}
	readTmpData := make([]*tmpStruct, 0, 8)
	if err = UnmarshalByFilePath(p, &readTmpData); err != nil {
		panic(err)
	}

	fieldInfos, err := Unmarshal2FieldInfoMap(p)
	if err != nil {
		panic(err)
	}
	content, _ := ioutil.ReadFile(p)
	fmt.Println(string(content))
	os.RemoveAll(filepath.Dir(p))

	fmt.Println(fieldInfos)
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

func TestMarshalTopType(t *testing.T) {
	data, err := Marshal("哈哈123")
	if err != nil {
		panic(err)
	}
	d, _ := ioutil.ReadFile(data)
	os.RemoveAll(data)
	fmt.Println(string(d))
}

func TestMarshalUserInfo(t *testing.T) {
	type InsideUserInfo struct {
		Id              int64    `json:"id,omitempty"`
		Name            string   `json:"name,omitempty"`
		Sex             int      `json:"sex,omitempty"`
		Age             int      `json:"age,omitempty"`
		Birthday        string   `json:"birthday,omitempty"`
		IdCode          string   `json:"idCode,omitempty"`
		Phone           string   `json:"phone,omitempty"`
		Email           string   `json:"email,omitempty"`
		CompanyName     string   `json:"companyName,omitempty"`
		JobName         string   `json:"jobName,omitempty"`
		PostsName       string   `json:"postsName,omitempty"`
		DeptName        string   `json:"deptName,omitempty"`
		DeptMain        bool     `json:"deptMain,omitempty"`
		UserName        string   `json:"userName,omitempty"`
		RegistryTime    string   `json:"registryTime,omitempty"`
		DataInTime      string   `json:"dataInTime,omitempty"`
		EndUpdateTime   string   `json:"endUpdateTime,omitempty"`
		EndLoginTime    string   `json:"endLoginTime,omitempty"`
		Icon            string   `json:"icon,omitempty"`
		BackgroundImage string   `json:"backgroundImage,omitempty"`
		BackgroundColor string   `json:"backgroundColor,omitempty"`
		SysFlagList     []string `json:"sysFlagList"`
		QueueHost       string   `json:"queueHost,omitempty"`
		QueuePort       int      `json:"queuePort,omitempty"`
		ServerPort      string   `json:"serverPort,omitempty"`
		Token           string   `json:"token,omitempty"`
	}

	tmpData := &InsideUserInfo{
		SysFlagList: []string{"hh", "tt"},
	}
	marshalFileStr, err := Marshal(tmpData)
	defer os.RemoveAll(marshalFileStr)
	if err != nil {
		panic(err)
	}

	data, _ := ioutil.ReadFile(marshalFileStr)
	fmt.Println(string(data))

}

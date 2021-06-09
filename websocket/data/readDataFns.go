package data

import (
	"bytes"
	"errors"
	"os"
	"strconv"
)

type readDataFns struct {
	*os.File
}

func (this *readDataFns) ReadLine() (string, error) {
	buff := &bytes.Buffer{}
	tmpBuf := make([]byte, 1, 1)
	for {
		_, err := this.Read(tmpBuf)
		if err != nil {
			return "", err
		}
		if tmpBuf[0] == newLine {
			return buff.String(), nil
		}
		buff.Write(tmpBuf)
	}
}

func (this *readDataFns) ReadFieldType() (FieldType, error) {
	line, err := this.ReadLine()
	if err != nil {
		return 0, err
	}
	fieldType, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return 0, errors.New("转换字段类型失败")
	}
	if fieldType < int64(FieldTypeString) || fieldType > int64(FieldTypeError) {
		return 0, errors.New("非法的字段类型")
	}
	return FieldType(fieldType), nil
}

func (this *readDataFns) ReadFieldLen() (int64, error) {
	line, err := this.ReadLine()
	if err != nil {
		return 0, err
	}
	tmpLen, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return 0, errors.New("转换长度失败")
	}

	return tmpLen, nil
}

func (this *readDataFns) BreakLine() error {
	tmpBuf := make([]byte, 1, 1)
	for {
		_, err := this.Read(tmpBuf)
		if err != nil {
			return err
		}
		if tmpBuf[0] == newLine {
			return nil
		}
	}
}

func (this *readDataFns) BreakLen(len int64) error {
	_, err := this.Seek(len, 1)
	return err
}

func (this *readDataFns) NowIndex() (int64, error) {
	return this.Seek(0, 1)
}

func newReadDataFns(f *os.File) *readDataFns {
	return &readDataFns{
		File: f,
	}
}

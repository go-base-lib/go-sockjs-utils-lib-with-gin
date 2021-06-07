package data

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

type fieldReflectInfo struct {
	value reflect.Value
	field reflect.StructField
}

const newLine = '\n'

func Marshal(v interface{}) (p string, err error) {
	tmpDir := filepath.Join(os.TempDir(), "devPlatform")
	_ = os.MkdirAll(tmpDir, 0777)
	tmpFile, err := ioutil.TempFile(tmpDir, "*")
	if err != nil {
		return "", errors.New("创建数据文件失败")
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()
	defer tmpFile.Close()
	writer := bufio.NewWriter(tmpFile)
	defer writer.Flush()
	return tmpFile.Name(), marshal2File(reflect.TypeOf(v), reflect.ValueOf(v), writer, true)
}

func marshal2File(rt reflect.Type, rv reflect.Value, writer *bufio.Writer, writeStructType bool) error {
	for {
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rv = rv.Elem()
			continue
		}
		break
	}

	if rt.Kind() == reflect.Struct {
		return marshalStruct(rt, rv, writer, writeStructType)
	} else if rt.Kind() == reflect.Slice {
		return marshalSlice(rt, rv, writer)
	} else if rt.Kind() != reflect.Map {
		return errors.New("不支持的类型")
	}

	mapRange := rv.MapRange()
	for mapRange.Next() {
		key := mapRange.Key()
		value := mapRange.Value()
		if key.Kind() != reflect.String {
			return errors.New("map结构，key只能是String")
		}

		writer.WriteString(key.String())
		writer.WriteRune(newLine)

		if err := marshalByFieldVal(value.Type(), value, writer, true); err != nil {
			return err
		}

	}

	return nil
}

func marshalSlice(rt reflect.Type, rv reflect.Value, writer *bufio.Writer) error {
	sliceLen := rv.Len()
	fieldTypeStr := strconv.FormatInt(int64(FieldTypeList), 10)
	writer.WriteString(fieldTypeStr)
	writer.WriteRune(newLine)
	writer.WriteString(strconv.FormatInt(int64(sliceLen), 10))
	writer.WriteRune(newLine)
	sonFieldType := rt.Elem()
	_, fieldTypeStr, err := marshalType2FieldType(sonFieldType)
	if err != nil {
		return err
	}

	writer.WriteString(fieldTypeStr)
	writer.WriteRune(newLine)
	for i := 0; i < sliceLen; i++ {
		rvi := rv.Index(i)
		if err := marshalByFieldVal(sonFieldType, rvi, writer, false); err != nil {
			return err
		}
	}
	writer.WriteRune(newLine)
	return nil
}

func marshalStruct(rt reflect.Type, rv reflect.Value, writer *bufio.Writer, writeType bool) error {

	fieldNum := rt.NumField()
	if writeType {
		writer.WriteString(strconv.FormatInt(int64(FieldTypeStruct), 10))
		writer.WriteRune(newLine)
	}

	writer.WriteString(strconv.FormatInt(int64(fieldNum), 10))
	writer.WriteRune(newLine)
	for i := 0; i < fieldNum; i++ {
		field := rt.Field(i)
		fieldVal := rv.Field(i)
		name := getInterfaceFieldName(&field)
		writer.WriteString(name)
		writer.WriteRune(newLine)
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
			fieldVal = fieldVal.Elem()
		}

		if err := marshalByFieldVal(t, fieldVal, writer, true); err != nil {
			return err
		}
	}
	return nil
}

func marshalByFieldVal(t reflect.Type, fieldVal reflect.Value, writer *bufio.Writer, writeType bool) error {
	fieldType, fieldTypeStr, err := marshalType2FieldType(t)
	if err != nil {
		return err
	}

	if writeType && fieldType != FieldTypeList && fieldType != FieldTypeStruct {
		writer.WriteString(fieldTypeStr)
		writer.WriteRune(newLine)
	}

	switch fieldType {
	case FieldTypeString:
		str := fieldVal.String()
		strLen := len(str)
		writer.WriteString(strconv.FormatInt(int64(strLen), 10))
		writer.WriteRune(newLine)
		writer.WriteString(str)
	case FieldTypeBool:
		b := fieldVal.Bool()
		writer.WriteString(strconv.FormatBool(b))
		writer.WriteRune(newLine)
	case FieldTypeInteger:
		iVal := fieldVal.Int()
		str := strconv.FormatInt(iVal, 10)
		writer.WriteString(str)
		writer.WriteRune(newLine)
	case FieldTypeDouble:
		fVal := fieldVal.Float()
		str := strconv.FormatFloat(fVal, 'g', 12, 64)
		writer.WriteString(str)
		writer.WriteRune(newLine)
	case FieldTypeStruct, FieldTypeList:
		return marshal2File(t, fieldVal, writer, writeType)
	default:
		return errors.New("错误的变量类型")
	}
	return nil
}

func marshalType2FieldType(rt reflect.Type) (FieldType, string, error) {
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	var fieldType FieldType
	switch rt.Kind() {
	case reflect.String:
		fieldType = FieldTypeString
	case reflect.Map, reflect.Struct:
		fieldType = FieldTypeStruct
	case reflect.Slice:
		fieldType = FieldTypeList
	case reflect.Bool:
		fieldType = FieldTypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldType = FieldTypeInteger
	case reflect.Float32, reflect.Float64:
		fieldType = FieldTypeDouble
	default:
		return 0, "", errors.New("未知的字段类型")
	}
	return fieldType, strconv.FormatInt(int64(fieldType), 10), nil
}

func UnmarshalByFilePath(s string, v interface{}) error {
	file, err := os.OpenFile(s, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	return Unmarshal(file, v)
}

func Unmarshal(f *os.File, v interface{}) error {
	defer f.Close()
	return unmarshalByBufIoReader(bufio.NewReader(f), v)
}

func unmarshalByBufIoReader(reader *bufio.Reader, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("请传入对象地址")
	}

	rt := reflect.TypeOf(v)

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
		rt = rt.Elem()
	}

	return unmarshal(val, rt, reader, true)
}

func unmarshal(val reflect.Value, rt reflect.Type, reader *bufio.Reader, readStructType bool) error {
	if rt.Kind() == reflect.Struct {
		return unmarshalStruct(reader, rt, val, readStructType)
	}

	if rt.Kind() == reflect.Slice {
		return unmarshalTopSlice(reader, rt, val)
	}

	if rt.Kind() != reflect.Map {
		return errors.New("未知的结构类型")
	}

	mapKey := rt.Key()
	if mapKey.Kind() != reflect.String {
		return errors.New("不支持的key类型")
	}

	if readStructType {
		fieldTypeStr, _, err := reader.ReadLine()
		if err != nil {
			return err
		}

		fieldTypeInt, err := strconv.ParseInt(string(fieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("转换命令码失败")
		}
		fieldType := FieldType(fieldTypeInt)
		if fieldType != FieldTypeStruct {
			return errors.New("非法的顶层结构")
		}
	}

	fieldNumStr, _, err := reader.ReadLine()
	if err != nil {
		return err
	}

	fieldNum, err := strconv.ParseInt(string(fieldNumStr), 10, 64)
	if err != nil {
		return errors.New("获取字段数量失败")
	}

	for val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	for i := int64(0); i < fieldNum; i++ {
		nameBytes, _, err := reader.ReadLine()
		if err == io.EOF {
			return nil
		}
		name := string(nameBytes)

		if err != nil {
			return errors.New("获取数据字段名称失败")
		}

		fieldTypeStr, _, err := reader.ReadLine()
		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(string(fieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		tmpV := reflect.New(rt.Elem()).Elem()
		if err = settingVal(fieldType, tmpV, reader); err != nil {
			return err
		}
		val.SetMapIndex(reflect.ValueOf(name), tmpV)
	}
	return nil
}

func getInterfaceFieldName(field *reflect.StructField) string {
	socketTag := field.Tag.Get("socket")
	if socketTag != "" {
		return socketTag
	}
	return field.Name
}

func unmarshalTopSlice(reader *bufio.Reader, rt reflect.Type, rv reflect.Value) error {
	for {
		fieldTypeStr, _, err := reader.ReadLine()
		if err == io.EOF {
			return nil
		}

		if len(fieldTypeStr) == 0 {
			continue
		}

		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(string(fieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		if fieldType != FieldTypeList {
			return errors.New("类型无法进行匹配")
		}

		settingVal(fieldType, rv, reader)
	}
}

func unmarshalStruct(reader *bufio.Reader, rt reflect.Type, rv reflect.Value, readStructType bool) error {
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if readStructType {
		fieldTypeStr, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		fieldTypeNum, err := strconv.ParseInt(string(fieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("获取数据类型失败")
		}
		if FieldType(fieldTypeNum) != FieldTypeStruct {
			return errors.New("非法的数据类型")
		}
	}

	fieldNumber := rt.NumField()
	tmpFieldMap := make(map[string]*fieldReflectInfo)
	for i := 0; i < fieldNumber; i++ {
		field := rt.Field(i)
		fieldVal := rv.Field(i)
		if !fieldVal.CanSet() {
			continue
		}
		fieldName := getInterfaceFieldName(&field)
		tmpFieldMap[fieldName] = &fieldReflectInfo{
			value: fieldVal,
			field: field,
		}
	}

	fieldLenStr, _, err := reader.ReadLine()
	if err != nil {
		return errors.New("获取字段个数失败")
	}
	fieldLen, err := strconv.ParseInt(string(fieldLenStr), 10, 64)
	if err != nil {
		return errors.New("转换字段个数失败")
	}

	settingFieldNum := int64(0)

	for {
		settingFieldNum += 1
		if settingFieldNum > fieldLen {
			return nil
		}
		name, _, err := reader.ReadLine()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return errors.New("获取数据字段名称失败")
		}

		fieldTypeStr, _, err := reader.ReadLine()
		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(string(fieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		info, ok := tmpFieldMap[string(name)]
		if !ok {
			if fieldType == FieldTypeString {
				dataLenStr, _, err := reader.ReadLine()
				if err != nil {
					return errors.New("获取数据长度失败")
				}

				dataLen, err := strconv.ParseInt(string(dataLenStr), 10, 64)
				if err != nil {
					return errors.New("转换数据长度失败")
				}
				_, err = reader.Discard(int(dataLen))
				if err != nil {
					return err
				}
			} else if fieldType == FieldTypeList {
				eleLenStr, _, err := reader.ReadLine()
				if err != nil {
					return err
				}

				eleLen, err := strconv.ParseInt(string(eleLenStr), 10, 64)
				if err != nil {
					return errors.New("读取数据长度失败")
				}
				if info.value.Elem().Kind() != reflect.String {
					for i := 0; i < int(eleLen); i++ {
						_, _, err = reader.ReadLine()
						if err != nil {
							return err
						}
					}
				} else {
					for i := 0; i < int(eleLen); i++ {
						lineStr, _, err := reader.ReadLine()
						if err != nil {
							return err
						}
						dataLen, err := strconv.ParseInt(string(lineStr), 10, 64)
						if err != nil {
							return errors.New("获取数据长度失败")
						}
						reader.Discard(int(dataLen))
					}
					_, _, err = reader.ReadLine()
					if err != nil {
						return err
					}
				}

			} else {
				_, _, err = reader.ReadLine()
				if err != nil {
					return err
				}
			}

			continue
		}

		if err = settingVal(fieldType, info.value, reader); err != nil {
			return err
		}
	}
}

func settingVal(fieldTpe FieldType, val reflect.Value, reader *bufio.Reader) error {
	t := val.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var fieldData reflect.Value
	switch fieldTpe {
	case FieldTypeString:
		data, err := readStringData(reader)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeInteger:
		data, err := readIntData(reader)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeDouble:
		double, err := readDouble(reader)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(double)
	case FieldTypeBool:
		data, err := readBool(reader)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeList:
		eleLenStr, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		eleLenInt64, err := strconv.ParseInt(string(eleLenStr), 10, 64)
		if err != nil {
			return errors.New("获取Slice长度失败")
		}

		eleLen := int(eleLenInt64)

		tmpFieldTypeStr, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		tmpFieldTypeNum, err := strconv.ParseInt(string(tmpFieldTypeStr), 10, 64)
		if err != nil {
			return errors.New("换换类型失败")
		}

		tmpFieldType := FieldType(tmpFieldTypeNum)

		if t.Kind() == reflect.Interface {
			t = reflect.SliceOf(t)
		}
		slice := reflect.MakeSlice(t, 0, eleLen)
		tmpElem := t.Elem()
		for tmpElem.Kind() == reflect.Ptr {
			tmpElem = tmpElem.Elem()
		}
		for i := 0; i < eleLen; i++ {

			tmpV := reflect.New(tmpElem)
			if err = settingVal(tmpFieldType, tmpV.Elem(), reader); err != nil {
				return err
			}

			if t.Elem().Kind() == reflect.Ptr {
				slice = reflect.Append(slice, tmpV)
			} else {
				slice = reflect.Append(slice, tmpV.Elem())
			}
		}

		val.Set(slice)

		if tmpFieldType == FieldTypeString {
			_, _, err = reader.ReadLine()
			if err != nil {
				return err
			}
		}
		return nil
	case FieldTypeStruct:
		var newT reflect.Value
		if t.Kind() == reflect.Interface {
			var tmpMap map[string]interface{}
			t = reflect.TypeOf(tmpMap)
			newT = reflect.MakeMap(t)
		} else {
			newT = reflect.New(t)
		}
		if val.Kind() == reflect.Ptr {
			val.Set(newT)
		}
		for newT.Kind() == reflect.Ptr {
			newT = newT.Elem()
		}
		if val.Kind() != reflect.Ptr {
			val.Set(newT)
		}
		return unmarshal(val, t, reader, false)
	default:
		return errors.New("获取字段类型失败")
	}

	val.Set(fieldData.Convert(val.Type()))
	return nil
}

func readBool(r *bufio.Reader) (bool, error) {
	tmpBuf, _, err := r.ReadLine()
	if err != nil {
		return false, errors.New("读取内容失败")
	}
	isTrue := true
	if "0" == string(tmpBuf) {
		isTrue = false
	}
	return isTrue, nil
}

func readDouble(r *bufio.Reader) (float64, error) {
	tmpBuf, _, err := r.ReadLine()
	if err != nil {
		return 0, errors.New("读取内容失败")
	}
	float, err := strconv.ParseFloat(string(tmpBuf), 64)
	if err != nil {
		return 0, errors.New("转换浮点数失败")
	}
	return float, nil
}
func readIntData(r *bufio.Reader) (int64, error) {
	tmpBuf, _, err := r.ReadLine()
	if err != nil {
		return 0, errors.New("读取内容失败")
	}
	parseInt, err := strconv.ParseInt(string(tmpBuf), 10, 64)
	if err != nil {
		return 0, errors.New("转换数字失败")
	}
	return parseInt, nil
}

func readStringData(r *bufio.Reader) (string, error) {
	dataLenStr, _, err := r.ReadLine()
	if err != nil {
		return "", errors.New("获取数据长度失败")
	}

	dataLen, err := strconv.ParseInt(string(dataLenStr), 10, 64)
	if err != nil {
		return "", errors.New("转换数据长度失败")
	}
	tmpBuf := make([]byte, dataLen, dataLen)
	_, err = r.Read(tmpBuf)
	if err != nil {
		return "", errors.New("读取内容失败")
	}
	return string(tmpBuf), nil
}

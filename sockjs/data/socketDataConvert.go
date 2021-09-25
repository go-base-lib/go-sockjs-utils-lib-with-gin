package data

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type fieldReflectInfo struct {
	value reflect.Value
	field reflect.StructField
}

type ErrorMsg struct {
	Code string
	Msg  string
}

func (e *ErrorMsg) Error() string {
	return e.Msg
}

const newLine = '\n'

func IsVoid(data string) bool {
	if len(data) == 0 {
		return false
	}

	return data[0] == strconv.FormatInt(int64(FieldTypeVoid), 10)[0]
}

func IsErr(data string) bool {
	if len(data) == 0 {
		return false
	}
	return data[0] == strconv.FormatInt(int64(FieldTypeError), 10)[0]
}

func Unmarshal2Err(f *os.File) (*ErrorMsg, error) {
	defer f.Close()
	fieldType, err := readLine(f)
	if err != nil {
		return nil, err
	}

	if fieldType != strconv.FormatInt(int64(FieldTypeError), 10) {
		return nil, errors.New("非错误类型")
	}

	code, err := readLine(f)
	if err != nil {
		return nil, err
	}

	msgLenStr, err := readLine(f)
	if err != nil {
		return nil, err
	}

	msgLen, err := strconv.ParseInt(msgLenStr, 10, 64)
	if err != nil {
		return nil, errors.New("转换错误消息长度失败")
	}

	msg, err := readLenStr(int(msgLen), f)
	if err != nil {
		return nil, err
	}

	return &ErrorMsg{
		Code: code,
		Msg:  msg,
	}, nil

}

func MarshalVoidStr() (string, error) {
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
	tmpFile.WriteString(strconv.FormatInt(int64(FieldTypeVoid), 10) + string(newLine))
	return tmpFile.Name(), nil
}

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

func MarshalErr(code, msg string) (string, error) {
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
	tmpFile.WriteString(strconv.FormatInt(int64(FieldTypeError), 10))
	tmpFile.WriteString("\n")
	tmpFile.WriteString(code)
	tmpFile.WriteString("\n")
	tmpFile.WriteString(strconv.FormatInt(int64(utf8.RuneCountInString(msg)), 10))
	tmpFile.WriteString("\n")
	tmpFile.WriteString(msg)
	return tmpFile.Name(), nil
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

	if rt.Kind() == reflect.Slice {
		return marshalSlice(rt, rv, writer)
	}

	if rt.Kind() == reflect.Struct {
		return marshalStruct(rt, rv, writer, writeStructType)
	}

	if rt.Kind() != reflect.Map {
		return marshalByFieldVal(rt, rv, writer, true)
	}

	fieldNum := rv.Len()
	writer.WriteString(strconv.FormatInt(int64(FieldTypeStruct), 10))
	writer.WriteRune(newLine)
	writer.WriteString(strconv.FormatInt(int64(fieldNum), 10))
	writer.WriteRune(newLine)
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
	if sliceLen == 0 {
		return nil
	}
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

	//writer.WriteString(strconv.FormatInt(int64(fieldNum), 10))
	//writer.WriteRune(newLine)
	tmpFieldMap := make(map[string]*fieldReflectInfo, fieldNum)
	for i := 0; i < fieldNum; i++ {
		field := rt.Field(i)
		fieldVal := rv.Field(i)
		name := getInterfaceFieldName(&field)
		if name == "" {
			continue
		}
		split := strings.Split(name, ",")
		marshalZero := true
		if len(split) == 2 {
			name = split[0]
			if split[1] == "omitempty" {
				marshalZero = false
			}
		}

		if !marshalZero && fieldVal.IsZero() {
			continue
		}
		tmpFieldMap[name] = &fieldReflectInfo{
			value: fieldVal,
			field: field,
		}

	}

	if len(tmpFieldMap) == 0 {
		writer.WriteString("0")
		writer.WriteRune(newLine)
		return nil
	}

	writer.WriteString(strconv.FormatInt(int64(len(tmpFieldMap)), 10))
	writer.WriteRune(newLine)
	for name, fieldInfo := range tmpFieldMap {
		field := fieldInfo.field
		fieldVal := fieldInfo.value
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
	for fieldVal.Kind() == reflect.Interface {
		fieldVal = fieldVal.Elem()
		t = fieldVal.Type()
	}
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
		strLen := utf8.RuneCountInString(str)
		writer.WriteString(fmt.Sprintln(strLen))
		writer.WriteString(str)
	case FieldTypeBool:
		fallthrough
	case FieldTypeInteger:
		fallthrough
	case FieldTypeDouble:
		writer.WriteString(fmt.Sprintln(fieldVal.Interface()))
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

func Unmarshal2FieldInfoMap(fp string) (map[string]*FieldInfo, error) {
	f, err := os.OpenFile(fp, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rFns := newReadDataFns(f)
	fieldType, err := rFns.ReadFieldType()
	if err != nil {
		return nil, err
	}
	return unmarshal2FieldInfoMap(rFns, fieldType)
}

func unmarshal2FieldInfoMap(r *readDataFns, fieldType FieldType) (map[string]*FieldInfo, error) {
	result := make(map[string]*FieldInfo)
	if fieldType == FieldTypeString || fieldType == FieldTypeInteger || fieldType == FieldTypeBool || fieldType == FieldTypeDouble {
		index, err := r.NowIndex()
		if err != nil {
			return nil, err
		}

		fieldInfo := &FieldInfo{
			name:     "0",
			startPos: index,
		}
		if fieldType == FieldTypeString {
			fieldLen, err := r.ReadFieldLen()
			if err != nil {
				return nil, err
			}
			fieldInfo.endPos = fieldLen + 1
		} else {
			if err = r.BreakLine(); err != nil {
				return nil, err
			}
			endPos, err := r.NowIndex()
			if err != nil {
				return nil, err
			}
			fieldInfo.endPos = endPos
		}
		result["0"] = fieldInfo
		return result, nil
	}
	if fieldType == FieldTypeList {
		listLen, err := r.ReadFieldLen()
		if err != nil {
			return nil, err
		}
		if listLen == 0 {
			return nil, err
		}
		eleFieldType, err := r.ReadFieldType()
		if err != nil {
			return nil, err
		}

		for i := int64(0); i < listLen; i++ {
			name := strconv.FormatInt(i, 10)
			startPos, err := r.NowIndex()
			if err != nil {
				return nil, err
			}

			if eleFieldType == FieldTypeStruct || eleFieldType == FieldTypeList {
				childrenMap, err := unmarshal2FieldInfoMap(r, eleFieldType)
				if err != nil {
					return nil, err
				}
				endPos, err := r.NowIndex()
				if err != nil {
					return nil, err
				}
				result[name] = &FieldInfo{
					name:     name,
					startPos: startPos,
					endPos:   endPos,
					len:      endPos - startPos,
					children: childrenMap,
				}
				continue
			}

			if eleFieldType == FieldTypeString {
				fieldLen, err := r.ReadFieldLen()
				if err != nil {
					return nil, err
				}
				if err = r.BreakLen(fieldLen); err != nil {
					return nil, err
				}
			} else {
				if err = r.BreakLine(); err != nil {
					return nil, err
				}
			}
			endPos, err := r.NowIndex()
			if err != nil {
				return nil, err
			}
			result[name] = &FieldInfo{
				name:     name,
				startPos: startPos,
				endPos:   endPos,
				len:      endPos - startPos,
			}
		}
		if eleFieldType == FieldTypeString {
			if err = r.BreakLine(); err != nil {
				return nil, err
			}
		}
		return result, nil
	}

	return unmarshalObj2FieldInfoMap(r, fieldType)
}

func unmarshalObj2FieldInfoMap(r *readDataFns, fieldType FieldType) (map[string]*FieldInfo, error) {
	if fieldType == FieldTypeList {
		return unmarshal2FieldInfoMap(r, fieldType)
	}

	if fieldType != FieldTypeStruct {
		return nil, errors.New("不支持非Obj或Array的顶层结构")
	}

	fieldNum, err := r.ReadFieldLen()
	if err != nil {
		return nil, err
	}

	if fieldNum == 0 {
		return nil, err
	}

	result := make(map[string]*FieldInfo)
	for i := int64(0); i < fieldNum; i++ {
		name, err := r.ReadLine()
		if err != nil {
			return nil, err
		}

		eleFieldType, err := r.ReadFieldType()
		if err != nil {
			return nil, err
		}

		startPos, err := r.NowIndex()
		if err != nil {
			return nil, err
		}

		if eleFieldType == FieldTypeStruct || eleFieldType == FieldTypeList {
			childrenMap, err := unmarshal2FieldInfoMap(r, eleFieldType)
			if err != nil {
				return nil, err
			}
			endPos, err := r.NowIndex()
			if err != nil {
				return nil, err
			}
			result[name] = &FieldInfo{
				name:     name,
				startPos: startPos,
				endPos:   endPos,
				len:      endPos - startPos,
				children: childrenMap,
			}
			continue
		}

		if eleFieldType == FieldTypeString {
			fieldLen, err := r.ReadFieldLen()
			if err != nil {
				return nil, err
			}
			if err = r.BreakLen(fieldLen); err != nil {
				return nil, err
			}
		} else {
			if err = r.BreakLine(); err != nil {
				return nil, err
			}
		}
		endPos, err := r.NowIndex()
		if err != nil {
			return nil, err
		}
		result[name] = &FieldInfo{
			name:     name,
			startPos: startPos,
			endPos:   endPos,
			len:      endPos - startPos,
		}
	}
	return result, err
}

func UnmarshalByFilePath(s string, v interface{}) error {
	file, err := os.OpenFile(s, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	return Unmarshal(file, v)
}

func Unmarshal(f *os.File, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("请传入对象地址")
	}

	rt := reflect.TypeOf(v)

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
		rt = rt.Elem()
	}

	return unmarshal(val, rt, f, true)
}

func unmarshal(val reflect.Value, rt reflect.Type, f *os.File, readStructType bool) error {
	fieldType, _, err := marshalType2FieldType(rt)
	if err != nil {
		return nil
	}

	if fieldType == FieldTypeString || fieldType == FieldTypeInteger || fieldType == FieldTypeBool || fieldType == FieldTypeDouble {
		_, err = readLine(f)
		if err != nil {
			return err
		}
		return settingVal(fieldType, val, f)
	}

	if rt.Kind() == reflect.Struct {
		return unmarshalStruct(f, rt, val, readStructType)
	}

	if rt.Kind() == reflect.Slice {
		return unmarshalTopSlice(f, val)
	}

	if rt.Kind() != reflect.Map {
		return errors.New("未知的结构类型")
	}

	mapKey := rt.Key()
	if mapKey.Kind() != reflect.String {
		return errors.New("不支持的key类型")
	}

	if readStructType {
		fieldTypeStr, err := readLine(f)
		if err != nil {
			return err
		}

		fieldTypeInt, err := strconv.ParseInt(fieldTypeStr, 10, 64)
		if err != nil {
			return errors.New("转换命令码失败")
		}
		fieldType := FieldType(fieldTypeInt)
		if fieldType != FieldTypeStruct {
			return errors.New("非法的顶层结构")
		}
	}

	fieldNumStr, err := readLine(f)
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
		name, err := readLine(f)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return errors.New("获取数据字段名称失败")
		}

		fieldTypeStr, err := readLine(f)
		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(fieldTypeStr, 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		tmpV := reflect.New(rt.Elem()).Elem()
		if err = settingVal(fieldType, tmpV, f); err != nil {
			return err
		}
		val.SetMapIndex(reflect.ValueOf(name), tmpV)
	}
	return nil
}

func getInterfaceFieldName(field *reflect.StructField) string {
	socketTag := field.Tag.Get("json")
	if socketTag != "" {
		if socketTag == "-" {
			return ""
		}
		return socketTag
	}
	return field.Name
}

func unmarshalTopSlice(f *os.File, rv reflect.Value) error {
	for {
		fieldTypeStr, err := readLine(f)
		if err == io.EOF {
			return nil
		}

		if len(fieldTypeStr) == 0 {
			continue
		}

		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(fieldTypeStr, 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		if fieldType != FieldTypeList {
			return errors.New("类型无法进行匹配")
		}

		settingVal(fieldType, rv, f)
	}
}

func unmarshalStruct(f *os.File, rt reflect.Type, rv reflect.Value, readStructType bool) error {
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if readStructType {
		fieldTypeStr, err := readLine(f)
		if err != nil {
			return err
		}
		fieldTypeNum, err := strconv.ParseInt(fieldTypeStr, 10, 64)
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
		if fieldName == "" {
			continue
		}

		fieldName = strings.Split(fieldName, ",")[0]

		tmpFieldMap[fieldName] = &fieldReflectInfo{
			value: fieldVal,
			field: field,
		}
	}

	fieldLenStr, err := readLine(f)
	if err != nil {
		return errors.New("获取字段个数失败")
	}
	fieldLen, err := strconv.ParseInt(fieldLenStr, 10, 64)
	if err != nil {
		return errors.New("转换字段个数失败")
	}

	settingFieldNum := int64(0)

	for {
		settingFieldNum += 1
		if settingFieldNum > fieldLen {
			return nil
		}
		name, err := readLine(f)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return errors.New("获取数据字段名称失败")
		}

		fieldTypeStr, err := readLine(f)
		if err != nil {
			return errors.New("获取字段类型失败")
		}

		fieldTypeInt, err := strconv.ParseInt(fieldTypeStr, 10, 64)
		if err != nil {
			return errors.New("转换字段类型失败")
		}

		fieldType := FieldType(fieldTypeInt)

		info, ok := tmpFieldMap[name]
		if !ok {
			if fieldType == FieldTypeString {
				dataLenStr, err := readLine(f)
				if err != nil {
					return errors.New("获取数据长度失败")
				}

				dataLen, err := strconv.ParseInt(dataLenStr, 10, 64)
				if err != nil {
					return errors.New("转换数据长度失败")
				}
				_, err = f.Seek(dataLen, 1)
				//_, err = reader.Discard(int(dataLen))
				if err != nil {
					return err
				}
			} else if fieldType == FieldTypeList {
				eleLenStr, err := readLine(f)
				if err != nil {
					return err
				}

				eleLen, err := strconv.ParseInt(eleLenStr, 10, 64)
				if err != nil {
					return errors.New("读取数据长度失败")
				}
				if info.value.Elem().Kind() != reflect.String {
					for i := 0; i < int(eleLen); i++ {
						_, err = readLine(f)
						if err != nil {
							return err
						}
					}
				} else {
					for i := 0; i < int(eleLen); i++ {
						lineStr, err := readLine(f)
						if err != nil {
							return err
						}
						dataLen, err := strconv.ParseInt(string(lineStr), 10, 64)
						if err != nil {
							return errors.New("获取数据长度失败")
						}
						f.Seek(dataLen, 1)
						//reader.Discard(int(dataLen))
					}
					_, err = readLine(f)
					if err != nil {
						return err
					}
				}

			} else {
				_, err = readLine(f)
				if err != nil {
					return err
				}
			}

			continue
		}

		if err = settingVal(fieldType, info.value, f); err != nil {
			return err
		}
	}
}

func settingVal(fieldTpe FieldType, val reflect.Value, f *os.File) error {
	t := val.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var fieldData reflect.Value
	switch fieldTpe {
	case FieldTypeString:
		data, err := readStringData(f)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeInteger:
		data, err := readIntData(f)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeDouble:
		double, err := readDouble(f)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(double)
	case FieldTypeBool:
		data, err := readBool(f)
		if err != nil {
			return err
		}
		fieldData = reflect.ValueOf(data)
	case FieldTypeList:
		eleLenStr, err := readLine(f)
		if err != nil {
			return err
		}
		eleLenInt64, err := strconv.ParseInt(eleLenStr, 10, 64)
		if err != nil {
			return errors.New("获取Slice长度失败")
		}

		eleLen := int(eleLenInt64)

		if eleLen == 0 {
			return nil
		}

		tmpFieldTypeStr, err := readLine(f)
		if err != nil {
			return err
		}
		tmpFieldTypeNum, err := strconv.ParseInt(tmpFieldTypeStr, 10, 64)
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
			if err = settingVal(tmpFieldType, tmpV.Elem(), f); err != nil {
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
			_, err = readLine(f)
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
		return unmarshal(val, t, f, false)
	default:
		return errors.New("获取字段类型失败")
	}

	val.Set(fieldData.Convert(val.Type()))
	return nil
}

func readBool(f *os.File) (bool, error) {
	tmpBuf, err := readLine(f)
	if err != nil {
		return false, errors.New("读取内容失败")
	}
	isTrue := true
	if "0" == string(tmpBuf) {
		isTrue = false
	}
	return isTrue, nil
}

func readDouble(f *os.File) (float64, error) {
	tmpBuf, err := readLine(f)
	if err != nil {
		return 0, errors.New("读取内容失败")
	}
	float, err := strconv.ParseFloat(string(tmpBuf), 64)
	if err != nil {
		return 0, errors.New("转换浮点数失败")
	}
	return float, nil
}
func readIntData(f *os.File) (int64, error) {
	tmpBuf, err := readLine(f)
	if err != nil {
		return 0, errors.New("读取内容失败")
	}
	parseInt, err := strconv.ParseInt(tmpBuf, 10, 64)
	if err != nil {
		return 0, errors.New("转换数字失败")
	}
	return parseInt, nil
}
func readStringData(f *os.File) (string, error) {
	dataLenStr, err := readLine(f)
	if err != nil {
		return "", errors.New("获取数据长度失败")
	}

	dataLen, err := strconv.ParseInt(dataLenStr, 10, 64)
	if err != nil {
		return "", errors.New("转换数据长度失败")
	}

	str, err := readLenStr(int(dataLen), f)
	return str, nil
}

func readLine(f *os.File) (string, error) {
	buff := &bytes.Buffer{}
	tmpBuf := make([]byte, 1, 1)
	for {
		_, err := f.Read(tmpBuf)
		if err == io.EOF && buff.Len() > 0 {
			return buff.String(), nil
		}
		if err != nil {
			return "", err
		}
		if tmpBuf[0] == newLine {
			return buff.String(), nil
		}
		buff.Write(tmpBuf)
	}
}

func readLenStr(length int, f *os.File) (string, error) {
	str := ""
	totalLen := 0
ReadStrStart:
	line, err := readLine(f)
	if err != nil {
		return "", err
	}
	lineRuneArr := []rune(line)
	totalLen += len(lineRuneArr)
	if totalLen < length {
		str += line + "\n"
		goto ReadStrStart
	}

	if totalLen > length {
		overLen := totalLen - length
		str += string(lineRuneArr[:length])
		_, err = f.Seek(-int64(overLen+1), 1)
		if err != nil {
			return "", err
		}
	}

	if totalLen == length {
		//f.Seek(-1, 1)
		str = line
	}

	return str, nil
}

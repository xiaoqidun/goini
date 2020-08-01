package goini

import (
	"bytes"
	"errors"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// GoINI GoINI数据结构
type GoINI struct {
	data        []byte
	dataMap     map[string]map[string]string
	nameList    []string
	dataKeyList map[string][]string
	commonField string
}

// NewGoINI 获取GoINI对象
func NewGoINI() *GoINI {
	return &GoINI{
		commonField: "BuiltCommon",
	}
}

// parse ini配置文件解析器
func (ini *GoINI) parse() {
	currentName := ini.commonField
	iniData := make(map[string]map[string]string)
	iniData[currentName] = make(map[string]string)
	ini.nameList = append(ini.nameList, currentName)
	ini.dataKeyList = make(map[string][]string)
	for _, v := range bytes.Split(ini.data, []byte("\n")) {
		line := bytes.TrimSpace(v)
		lineLen := len(line)
		if line == nil || lineLen < 3 ||
			line[0] == 35 || line[0] == 59 {
			continue
		}
		if line[0] == 91 && line[lineLen-1] == 93 {
			name := bytes.TrimSpace(line[1 : lineLen-1])
			if name != nil {
				currentName = string(name)
				if _, ok := iniData[currentName]; !ok {
					iniData[currentName] = make(map[string]string)
					ini.nameList = append(ini.nameList, currentName)
				}
			}
			continue
		}
		split := bytes.IndexByte(line, 61)
		if split == -1 {
			split = bytes.IndexByte(line, 32)
			if split == -1 {
				continue
			}
		}
		key := bytes.TrimSpace(line[0:split])
		value := bytes.TrimSpace(line[split+1:])
		valueLen := len(value)
		keyStr, valueStr := string(key), string(value)
		if keyStr == "" {
			continue
		}
		if _, ok := iniData[currentName][keyStr]; ok {
			continue
		}
		ini.dataKeyList[currentName] = append(ini.dataKeyList[currentName], keyStr)
		if value == nil {
			iniData[currentName][keyStr] = ""
			continue
		}
		if valueLen >= 2 &&
			((value[0] == 34 && value[valueLen-1] == 34) ||
				(value[0] == 39 && value[valueLen-1] == 39) ||
				(value[0] == 96 && value[valueLen-1] == 96)) {
			iniData[currentName][keyStr] = string(value[1 : valueLen-1])
			continue
		}
		iniData[currentName][keyStr] = valueStr
	}
	ini.dataMap = iniData
}

// String 获取GoINI对象字符串形式
func (ini *GoINI) String() string {
	var iniLines []string
	for _, name := range ini.GetNames("") {
		if name != ini.commonField {
			if len(iniLines) < 1 {
				iniLines = append(iniLines, "["+name+"]")
			} else {
				iniLines = append(iniLines, "\n["+name+"]")
			}
		}
		for _, key := range ini.GetNameKeys(name, "") {
			nameValue := ini.GetString(name, key, "")
			if ok, _ := regexp.MatchString("^\\s|\\s$", nameValue); ok {
				iniLines = append(iniLines, key+" = \""+nameValue+"\"")
				continue
			}
			nameValueLen := len(nameValue)
			if nameValueLen >= 2 &&
				((nameValue[0] == 34 && nameValue[nameValueLen-1] == 34) ||
					(nameValue[0] == 39 && nameValue[nameValueLen-1] == 39) ||
					(nameValue[0] == 96 && nameValue[nameValueLen-1] == 96)) {
				tag := `"`
				if nameValue[0] == 34 {
					tag = "`"
				}
				iniLines = append(iniLines, key+" = "+tag+nameValue+tag)
			} else {
				iniLines = append(iniLines, key+" = "+nameValue)
			}
		}
	}
	return strings.Join(iniLines, "\n")
}

// SetData 从代码读取配置并解析
func (ini *GoINI) SetData(fileData []byte) {
	ini.data = bytes.TrimSpace(fileData)
	ini.parse()
}

// LoadFile 从文件读取配置并解析
func (ini *GoINI) LoadFile(fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	ini.data = bytes.TrimSpace(b)
	ini.parse()
	return nil
}

// GetNames 获取配置文件分区列表
func (ini *GoINI) GetNames(match string) []string {
	if match == "" {
		return ini.nameList
	}
	var matchNameList []string
	for _, name := range ini.nameList {
		if b, e := regexp.MatchString(match, name); e == nil && b {
			matchNameList = append(matchNameList, name)
		}
	}
	return matchNameList
}

// GetNameKeys 获取分区下配置项列表
func (ini *GoINI) GetNameKeys(name string, match string) []string {
	var keyList []string
	if name == "" {
		name = ini.commonField
	}
	if keyList, ok := ini.dataKeyList[name]; ok {
		if match == "" {
			return keyList
		}
		var matchKeyList []string
		for _, key := range keyList {
			if b, e := regexp.MatchString(match, key); e == nil && b {
				matchKeyList = append(matchKeyList, key)
			}
		}
		return matchKeyList
	}
	return keyList
}

// GetBool 获取单个配置项的布尔值
func (ini *GoINI) GetBool(name string, key string, value bool) bool {
	boolStr := ini.GetString(name, key, "")
	switch strings.ToLower(boolStr) {
	case "y", "yes", "on":
		return true
	case "n", "no", "off":
		return false
	}
	if b, e := strconv.ParseBool(boolStr); e == nil {
		return b
	}
	return value
}

// GetInt64 获取单个配置项的数字值
func (ini *GoINI) GetInt64(name string, key string, value int64) int64 {
	int64Str := ini.GetString(name, key, "")
	if i, e := strconv.ParseInt(int64Str, 10, 64); e == nil {
		return i
	}
	return value
}

// GetString 获取单个配置项的字符值
func (ini *GoINI) GetString(name string, key string, value string) string {
	if 0 == len(name) {
		name = ini.commonField
	}
	if v, ok := ini.dataMap[name]; ok {
		if vv, ok := v[key]; ok {
			return vv
		}
	}
	return value
}

// GetFloat64 获取单个配置项的小数值
func (ini *GoINI) GetFloat64(name string, key string, value float64) float64 {
	float64Str := ini.GetString(name, key, "")
	if f, e := strconv.ParseFloat(float64Str, 64); e == nil {
		return f
	}
	return value
}

// MapToStruct 将配置映射到一个结构体
func (ini *GoINI) MapToStruct(ptr interface{}) (err error) {
	t := reflect.TypeOf(ptr)
	v := reflect.ValueOf(ptr)
	if t.Kind() != reflect.Ptr {
		err = errors.New("input struct ptr")
		return
	} else {
		t = t.Elem()
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		if !v.CanInterface() {
			continue
		}
		k := t.Field(i).Tag.Get("goini")
		if k == "" {
			k = t.Field(i).Name
		}
		switch v.Field(i).Kind() {
		case reflect.Struct:
			tt := v.Field(i).Type()
			vv := v.Field(i)
			for ii := 0; ii < tt.NumField(); ii++ {
				if !v.CanInterface() {
					continue
				}
				kk := tt.Field(ii).Tag.Get("goini")
				if kk == "" {
					kk = t.Field(ii).Name
				}
				switch vv.Field(ii).Kind() {
				case reflect.Bool:
					vv.Field(ii).SetBool(ini.GetBool(k, kk, false))
				case reflect.String:
					vv.Field(ii).SetString(ini.GetString(k, kk, ""))
				case reflect.Float32, reflect.Float64:
					vv.Field(ii).SetFloat(ini.GetFloat64(k, kk, 0))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					vv.Field(ii).SetInt(ini.GetInt64(k, kk, 0))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					vv.Field(ii).SetUint(uint64(ini.GetInt64(k, kk, 0)))
				}
			}
		case reflect.Bool:
			v.Field(i).SetBool(ini.GetBool(ini.commonField, k, false))
		case reflect.String:
			v.Field(i).SetString(ini.GetString(ini.commonField, k, ""))
		case reflect.Float32, reflect.Float64:
			v.Field(i).SetFloat(ini.GetFloat64(ini.commonField, k, 0))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.Field(i).SetInt(ini.GetInt64(ini.commonField, k, 0))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.Field(i).SetUint(uint64(ini.GetInt64(ini.commonField, k, 0)))
		}
	}
	return
}

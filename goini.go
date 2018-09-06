package goini

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

type GoINI struct {
	data        []byte
	dataMap     map[string]map[string]string
	nameList    []string
	dataKeyList map[string][]string
	commonField string
}

func NewGoINI() GoINI {
	return GoINI{
		commonField: "BuiltCommon",
	}
}
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
func (ini *GoINI) SetData(fileData []byte) {
	ini.data = bytes.TrimSpace(fileData)
}
func (ini *GoINI) LoadFile(fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	ini.data = bytes.TrimSpace(b)
	return nil
}
func (ini *GoINI) ParseIni() {
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
func (ini *GoINI) GetInt64(name string, key string, value int64) int64 {
	int64Str := ini.GetString(name, key, "")
	if i, e := strconv.ParseInt(int64Str, 10, 64); e == nil {
		return i
	}
	return value
}
func (ini *GoINI) GetString(name string, key string, value string) string {
	if 0 == len(name) {
		name = ini.commonField
	}
	if pos := strings.Index(key, "."); pos != -1 {
		name = key[:pos]
		key = key[pos+1:]
	}
	if v, ok := ini.dataMap[name]; ok {
		if vv, ok := v[key]; ok {
			return vv
		}
	}
	return value
}
func (ini *GoINI) GetFloat64(name string, key string, value float64) float64 {
	float64Str := ini.GetString(name, key, "")
	if f, e := strconv.ParseFloat(float64Str, 64); e == nil {
		return f
	}
	return value
}

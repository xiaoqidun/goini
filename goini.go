package goini

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	defaultTag         = "goini"
	defaultSeparator   = " = "
	defaultRootSection = "BuiltCommon"
)

var (
	errFilenameRequired      = errors.New("filename required")
	errStructPointerRequired = errors.New("struct pointer required")
)

// GoINI 表示INI配置解析器
type GoINI struct {
	data                  []byte
	filename              string
	fileLines             []iniLine
	structTag             string
	rootSection           string
	sectionKeys           map[string][]string
	sectionOrder          []string
	sectionValues         map[string]map[string]string
	sectionLineIndexes    map[string]int
	sectionKeyLineIndexes map[string]map[string]int
}

// NewGoINI 创建GoINI对象
// 返回: *GoINI GoINI对象
func NewGoINI() *GoINI {
	return &GoINI{structTag: defaultTag, rootSection: defaultRootSection}
}

// SetData 从字节内容读取配置并解析
// 入参: fileData 配置文件内容
func (ini *GoINI) SetData(fileData []byte) {
	ini.filename = ""
	ini.data = append(ini.data[:0], fileData...)
	ini.parse()
}

// LoadFile 从文件读取配置并解析
// 入参: filename 配置文件路径
// 返回: error 错误信息
func (ini *GoINI) LoadFile(filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	ini.filename = filename
	ini.data = append(ini.data[:0], b...)
	ini.parse()
	return nil
}

// Save 将配置写回LoadFile读取的文件
// 返回: error 错误信息
func (ini *GoINI) Save() error {
	if ini.filename == "" {
		return errFilenameRequired
	}
	return ini.SaveFile(ini.filename)
}

// SaveFile 将配置写入指定文件
// 入参: filename 配置文件路径
// 返回: error 错误信息
func (ini *GoINI) SaveFile(filename string) error {
	if filename == "" {
		return errFilenameRequired
	}
	if err := os.WriteFile(filename, []byte(ini.String()), 0o666); err != nil {
		return err
	}
	ini.filename = filename
	return nil
}

// String 获取GoINI对象的字符串形式
// 返回: string 配置文件内容
func (ini *GoINI) String() string {
	renderedLines := make([]string, len(ini.fileLines))
	for i, line := range ini.fileLines {
		renderedLines[i] = renderLine(line)
	}
	return strings.Join(renderedLines, "\n")
}

// SetString 设置单个配置项的字符串值
// 入参: name 分区名称, key 配置项名称, value 配置项值
func (ini *GoINI) SetString(name string, key string, value string) {
	ini.ensureParsed()
	name = ini.normalizeSection(name)
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if index, ok := ini.findSectionKeyLineIndex(name, key); ok {
		ini.fileLines[index].keyValue = value
		ini.fileLines[index].modified = true
		ini.rebuildIndexes()
		return
	}
	ini.ensureSectionLine(name)
	ini.insertLine(ini.findSectionInsertIndex(name), iniLine{kind: lineKeyValue, sectionName: name, keyName: key, keyValue: value, separator: defaultSeparator})
}

// SetBool 设置单个配置项的布尔值
// 入参: name 分区名称, key 配置项名称, value 配置项值
func (ini *GoINI) SetBool(name string, key string, value bool) {
	ini.SetString(name, key, strconv.FormatBool(value))
}

// SetInt64 设置单个配置项的整数值
// 入参: name 分区名称, key 配置项名称, value 配置项值
func (ini *GoINI) SetInt64(name string, key string, value int64) {
	ini.SetString(name, key, strconv.FormatInt(value, 10))
}

// SetFloat64 设置单个配置项的浮点数值
// 入参: name 分区名称, key 配置项名称, value 配置项值
func (ini *GoINI) SetFloat64(name string, key string, value float64) {
	ini.SetString(name, key, strconv.FormatFloat(value, 'f', -1, 64))
}

// SetComment 设置配置项注释，空注释会删除已有注释
// 入参: name 分区名称, key 配置项名称, comment 注释内容
// 返回: bool 是否设置成功
func (ini *GoINI) SetComment(name string, key string, comment string) bool {
	if key == "" {
		return ini.SetSectionComment(name, comment)
	}
	ini.ensureParsed()
	name = ini.normalizeSection(name)
	key = strings.TrimSpace(key)
	index, ok := ini.findSectionKeyLineIndex(name, key)
	if !ok {
		return false
	}
	ini.replaceCommentBefore(index, comment)
	return true
}

// SetSectionComment 设置分区注释，空注释会删除已有注释
// 入参: name 分区名称, comment 注释内容
// 返回: bool 是否设置成功
func (ini *GoINI) SetSectionComment(name string, comment string) bool {
	ini.ensureParsed()
	name = ini.normalizeSection(name)
	if name == ini.rootSection {
		ini.replaceCommentBefore(ini.firstRootLineIndex(), comment)
		return true
	}
	ini.ensureSectionLine(name)
	index, ok := ini.findSectionLineIndex(name)
	if !ok {
		return false
	}
	ini.replaceCommentBefore(index, comment)
	return true
}

// GetNames 获取配置文件分区列表
// 入参: match 分区名称正则表达式
// 返回: []string 分区名称列表
func (ini *GoINI) GetNames(match string) []string {
	return matchStrings(ini.sectionOrder, match)
}

// GetNameKeys 获取分区下配置项列表
// 入参: name 分区名称, match 配置项名称正则表达式
// 返回: []string 配置项名称列表
func (ini *GoINI) GetNameKeys(name string, match string) []string {
	name = ini.normalizeSection(name)
	return matchStrings(ini.sectionKeys[name], match)
}

// GetString 获取单个配置项的字符串值
// 入参: name 分区名称, key 配置项名称, value 默认值
// 返回: string 配置项值
func (ini *GoINI) GetString(name string, key string, value string) string {
	name = ini.normalizeSection(name)
	if v, ok := ini.sectionValues[name]; ok {
		if vv, ok := v[key]; ok {
			return vv
		}
	}
	return value
}

// GetBool 获取单个配置项的布尔值
// 入参: name 分区名称, key 配置项名称, value 默认值
// 返回: bool 配置项值
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

// GetInt64 获取单个配置项的整数值
// 入参: name 分区名称, key 配置项名称, value 默认值
// 返回: int64 配置项值
func (ini *GoINI) GetInt64(name string, key string, value int64) int64 {
	int64Str := ini.GetString(name, key, "")
	if i, e := strconv.ParseInt(int64Str, 10, 64); e == nil {
		return i
	}
	return value
}

// GetFloat64 获取单个配置项的浮点数值
// 入参: name 分区名称, key 配置项名称, value 默认值
// 返回: float64 配置项值
func (ini *GoINI) GetFloat64(name string, key string, value float64) float64 {
	float64Str := ini.GetString(name, key, "")
	if f, e := strconv.ParseFloat(float64Str, 64); e == nil {
		return f
	}
	return value
}

// GetComment 获取配置项注释
// 入参: name 分区名称, key 配置项名称
// 返回: string 注释内容
func (ini *GoINI) GetComment(name string, key string) string {
	if key == "" {
		return ini.GetSectionComment(name)
	}
	ini.ensureParsed()
	name = ini.normalizeSection(name)
	index, ok := ini.findSectionKeyLineIndex(name, strings.TrimSpace(key))
	if !ok {
		return ""
	}
	return ini.commentBefore(index)
}

// GetSectionComment 获取分区注释
// 入参: name 分区名称
// 返回: string 注释内容
func (ini *GoINI) GetSectionComment(name string) string {
	ini.ensureParsed()
	name = ini.normalizeSection(name)
	if name == ini.rootSection {
		return ini.commentBefore(ini.firstRootLineIndex())
	}
	index, ok := ini.findSectionLineIndex(name)
	if !ok {
		return ""
	}
	return ini.commentBefore(index)
}

// SetTag 设置结构体的tag键名称
// 入参: tag tag键名称
func (ini *GoINI) SetTag(tag string) {
	ini.structTag = tag
}

// MapToStruct 将配置映射到一个结构体
// 入参: ptr 结构体指针
// 返回: error 错误信息
func (ini *GoINI) MapToStruct(ptr interface{}) error {
	v := reflect.ValueOf(ptr)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return errStructPointerRequired
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errStructPointerRequired
	}
	ini.mapStruct(v)
	return nil
}

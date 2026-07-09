package goini

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type lineKind int

const (
	lineBlank lineKind = iota
	lineComment
	lineSection
	lineKeyValue
	lineRaw
)

type iniLine struct {
	kind        lineKind
	text        string
	sectionName string
	keyName     string
	keyValue    string
	indent      string
	separator   string
	modified    bool
}

type keyValueParts struct {
	key       string
	value     string
	indent    string
	separator string
}

func (ini *GoINI) parse() {
	ini.fileLines = nil
	currentSection := ini.rootSection
	if len(ini.data) == 0 {
		ini.rebuildIndexes()
		return
	}
	for _, rawLine := range strings.Split(string(ini.data), "\n") {
		line, section := parseRawLine(rawLine, currentSection)
		currentSection = section
		ini.fileLines = append(ini.fileLines, line)
	}
	ini.rebuildIndexes()
}

func parseRawLine(rawLine string, sectionName string) (iniLine, string) {
	rawLine = strings.TrimSuffix(rawLine, "\r")
	line := iniLine{kind: lineRaw, text: rawLine, sectionName: sectionName}
	trimmed := strings.TrimSpace(rawLine)
	if trimmed == "" {
		line.kind = lineBlank
		return line, sectionName
	}
	if isComment(trimmed) {
		line.kind = lineComment
		return line, sectionName
	}
	if nextSection, ok := parseSection(trimmed); ok {
		line.kind = lineSection
		line.sectionName = nextSection
		return line, nextSection
	}
	if parts, ok := parseKeyValue(rawLine); ok {
		line.kind = lineKeyValue
		line.keyName = parts.key
		line.keyValue = trimSurroundingQuotes(parts.value)
		line.indent = parts.indent
		line.separator = parts.separator
		return line, sectionName
	}
	return line, sectionName
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";")
}

func parseSection(line string) (string, bool) {
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return "", false
	}
	name := strings.TrimSpace(line[1 : len(line)-1])
	return name, name != ""
}

func parseKeyValue(line string) (keyValueParts, bool) {
	indentLen := leadingSpaceLen(line)
	indent := line[:indentLen]
	body := line[indentLen:]
	if split := strings.IndexByte(body, '='); split >= 0 {
		return parseKeyValueParts(body, split, split+1, indent)
	}
	if split := strings.IndexFunc(body, unicode.IsSpace); split >= 0 {
		return parseKeyValueParts(body, split, split, indent)
	}
	return keyValueParts{}, false
}

func parseKeyValueParts(body string, keyEnd int, valueStart int, indent string) (keyValueParts, bool) {
	keyPart := body[:keyEnd]
	valuePart := body[valueStart:]
	key := strings.TrimSpace(keyPart)
	if key == "" {
		return keyValueParts{}, false
	}
	separatorStart := len(strings.TrimRightFunc(keyPart, unicode.IsSpace))
	separatorEnd := valueStart + leadingSpaceLen(valuePart)
	return keyValueParts{
		key:       key,
		value:     strings.TrimSpace(valuePart),
		indent:    indent,
		separator: body[separatorStart:separatorEnd],
	}, true
}

func leadingSpaceLen(text string) int {
	return len(text) - len(strings.TrimLeftFunc(text, unicode.IsSpace))
}

func trimSurroundingQuotes(value string) string {
	if !hasSurroundingQuotes(value) {
		return value
	}
	return value[1 : len(value)-1]
}

func hasSurroundingQuotes(value string) bool {
	if len(value) < 2 {
		return false
	}
	first, last := value[0], value[len(value)-1]
	return first == last && (first == '"' || first == '\'' || first == '`')
}

func (ini *GoINI) rebuildIndexes() {
	ini.sectionKeys = make(map[string][]string)
	ini.sectionOrder = nil
	ini.sectionValues = make(map[string]map[string]string)
	ini.sectionLineIndexes = make(map[string]int)
	ini.sectionKeyLineIndexes = make(map[string]map[string]int)
	ini.addSection(ini.rootSection)
	for i, line := range ini.fileLines {
		switch line.kind {
		case lineSection:
			ini.addSectionLineIndex(line.sectionName, i)
		case lineKeyValue:
			ini.addSectionKey(line.sectionName, line.keyName, line.keyValue, i)
		}
	}
}

func (ini *GoINI) addSection(sectionName string) {
	if _, ok := ini.sectionValues[sectionName]; ok {
		return
	}
	ini.sectionValues[sectionName] = make(map[string]string)
	ini.sectionOrder = append(ini.sectionOrder, sectionName)
}

func (ini *GoINI) addSectionLineIndex(sectionName string, lineIndex int) {
	if _, ok := ini.sectionLineIndexes[sectionName]; !ok {
		ini.sectionLineIndexes[sectionName] = lineIndex
	}
	ini.addSection(sectionName)
}

func (ini *GoINI) addSectionKey(sectionName string, keyName string, keyValue string, lineIndex int) {
	ini.addSection(sectionName)
	if _, ok := ini.sectionValues[sectionName][keyName]; ok {
		return
	}
	ini.sectionValues[sectionName][keyName] = keyValue
	ini.sectionKeys[sectionName] = append(ini.sectionKeys[sectionName], keyName)
	if ini.sectionKeyLineIndexes[sectionName] == nil {
		ini.sectionKeyLineIndexes[sectionName] = make(map[string]int)
	}
	ini.sectionKeyLineIndexes[sectionName][keyName] = lineIndex
}

func (ini *GoINI) ensureParsed() {
	if ini.sectionValues == nil {
		ini.parse()
	}
}

func (ini *GoINI) normalizeSection(sectionName string) string {
	if sectionName == "" {
		return ini.rootSection
	}
	return sectionName
}

func (ini *GoINI) findSectionLineIndex(sectionName string) (int, bool) {
	index, ok := ini.sectionLineIndexes[sectionName]
	return index, ok
}

func (ini *GoINI) findSectionKeyLineIndex(sectionName string, keyName string) (int, bool) {
	if ini.sectionKeyLineIndexes[sectionName] == nil {
		return 0, false
	}
	index, ok := ini.sectionKeyLineIndexes[sectionName][keyName]
	return index, ok
}

func (ini *GoINI) ensureSectionLine(sectionName string) {
	if sectionName == ini.rootSection {
		ini.addSection(sectionName)
		return
	}
	if _, ok := ini.sectionValues[sectionName]; ok {
		return
	}
	if len(ini.fileLines) > 0 && renderLine(ini.fileLines[len(ini.fileLines)-1]) != "" {
		ini.fileLines = append(ini.fileLines, iniLine{kind: lineBlank})
	}
	ini.fileLines = append(ini.fileLines, iniLine{kind: lineSection, sectionName: sectionName, text: "[" + sectionName + "]"})
	ini.rebuildIndexes()
}

func (ini *GoINI) findSectionInsertIndex(sectionName string) int {
	if sectionName == ini.rootSection {
		return ini.findRootInsertIndex()
	}
	current := ini.rootSection
	insert := len(ini.fileLines)
	for i, line := range ini.fileLines {
		if line.kind == lineSection {
			current = line.sectionName
			if current == sectionName {
				insert = i + 1
			}
			continue
		}
		if current == sectionName {
			insert = i + 1
		}
	}
	return insert
}

func (ini *GoINI) findRootInsertIndex() int {
	for i, line := range ini.fileLines {
		if line.kind == lineSection {
			return i
		}
	}
	return len(ini.fileLines)
}

func (ini *GoINI) insertLine(index int, line iniLine) {
	if index < 0 || index > len(ini.fileLines) {
		index = len(ini.fileLines)
	}
	ini.fileLines = append(ini.fileLines, iniLine{})
	copy(ini.fileLines[index+1:], ini.fileLines[index:])
	ini.fileLines[index] = line
	ini.rebuildIndexes()
}

func (ini *GoINI) replaceLines(start int, end int, replacement []iniLine) {
	next := make([]iniLine, 0, len(ini.fileLines)-(end-start)+len(replacement))
	next = append(next, ini.fileLines[:start]...)
	next = append(next, replacement...)
	next = append(next, ini.fileLines[end:]...)
	ini.fileLines = next
	ini.rebuildIndexes()
}

func (ini *GoINI) replaceCommentBefore(index int, comment string) {
	start, end := ini.commentRangeBefore(index)
	ini.replaceLines(start, end, commentLines(comment))
}

func (ini *GoINI) commentRangeBefore(index int) (int, int) {
	start := index
	for start > 0 && ini.fileLines[start-1].kind == lineComment {
		start--
	}
	return start, index
}

func (ini *GoINI) commentBefore(index int) string {
	start, end := ini.commentRangeBefore(index)
	comments := make([]string, 0, end-start)
	for _, line := range ini.fileLines[start:end] {
		comments = append(comments, stripComment(line.text))
	}
	return strings.Join(comments, "\n")
}

func (ini *GoINI) firstRootLineIndex() int {
	for i, line := range ini.fileLines {
		if line.kind == lineKeyValue && line.sectionName == ini.rootSection {
			return i
		}
		if line.kind == lineSection {
			return i
		}
	}
	return 0
}

func commentLines(comment string) []iniLine {
	if comment == "" {
		return nil
	}
	parts := strings.Split(strings.ReplaceAll(comment, "\r\n", "\n"), "\n")
	lines := make([]iniLine, 0, len(parts))
	for _, part := range parts {
		lines = append(lines, iniLine{kind: lineComment, text: formatComment(part)})
	}
	return lines
}

func formatComment(comment string) string {
	trimmed := strings.TrimLeftFunc(comment, unicode.IsSpace)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
		return comment
	}
	if comment == "" {
		return "#"
	}
	return "# " + comment
}

func stripComment(comment string) string {
	trimmed := strings.TrimSpace(comment)
	if !isComment(trimmed) {
		return trimmed
	}
	trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "#"), ";")
	return strings.TrimPrefix(trimmed, " ")
}

func renderLine(line iniLine) string {
	switch line.kind {
	case lineSection:
		if line.text != "" && !line.modified {
			return line.text
		}
		return "[" + line.sectionName + "]"
	case lineKeyValue:
		if line.text != "" && !line.modified {
			return line.text
		}
		return renderKeyValue(line)
	default:
		return line.text
	}
}

func renderKeyValue(line iniLine) string {
	separator := line.separator
	if separator == "" {
		separator = defaultSeparator
	}
	return line.indent + line.keyName + separator + formatValue(line.keyValue)
}

func formatValue(value string) string {
	if strings.TrimSpace(value) == value && !hasSurroundingQuotes(value) {
		return value
	}
	quote := `"`
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		quote = "`"
	}
	return quote + value + quote
}

func matchStrings(items []string, match string) []string {
	if match == "" {
		return cloneStrings(items)
	}
	re, err := regexp.Compile(match)
	if err != nil {
		return nil
	}
	matches := make([]string, 0, len(items))
	for _, item := range items {
		if re.MatchString(item) {
			matches = append(matches, item)
		}
	}
	return matches
}

func cloneStrings(items []string) []string {
	if items == nil {
		return nil
	}
	cloned := make([]string, len(items))
	copy(cloned, items)
	return cloned
}

func (ini *GoINI) mapStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		name, ok := ini.fieldName(t.Field(i))
		if !ok {
			continue
		}
		if field.Kind() == reflect.Struct {
			ini.mapSection(field, name)
			continue
		}
		ini.setField(field, ini.rootSection, name)
	}
}

func (ini *GoINI) mapSection(v reflect.Value, section string) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() || field.Kind() == reflect.Struct {
			continue
		}
		name, ok := ini.fieldName(t.Field(i))
		if !ok {
			continue
		}
		ini.setField(field, section, name)
	}
}

func (ini *GoINI) fieldName(field reflect.StructField) (string, bool) {
	if ini.structTag != "" {
		if tag, ok := field.Tag.Lookup(ini.structTag); ok {
			name := strings.SplitN(tag, ",", 2)[0]
			if name == "-" {
				return "", false
			}
			if name != "" {
				return name, true
			}
		}
	}
	return field.Name, true
}

func (ini *GoINI) setField(field reflect.Value, section string, key string) {
	switch field.Kind() {
	case reflect.Bool:
		field.SetBool(ini.GetBool(section, key, false))
	case reflect.String:
		field.SetString(ini.GetString(section, key, ""))
	case reflect.Float32, reflect.Float64:
		field.SetFloat(ini.parseFloat(section, key, field.Type().Bits()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(ini.parseInt(section, key, field.Type().Bits()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(ini.parseUint(section, key, field.Type().Bits()))
	}
}

func (ini *GoINI) parseFloat(section string, key string, bitSize int) float64 {
	value, err := strconv.ParseFloat(ini.GetString(section, key, ""), bitSize)
	if err != nil {
		return 0
	}
	return value
}

func (ini *GoINI) parseInt(section string, key string, bitSize int) int64 {
	value, err := strconv.ParseInt(ini.GetString(section, key, ""), 10, bitSize)
	if err != nil {
		return 0
	}
	return value
}

func (ini *GoINI) parseUint(section string, key string, bitSize int) uint64 {
	value, err := strconv.ParseUint(ini.GetString(section, key, ""), 10, bitSize)
	if err != nil {
		return 0
	}
	return value
}

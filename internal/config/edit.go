package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/pelletier/go-toml/v2/unstable"

	"github.com/UsingCoding/bmx/internal/model"
)

type groupSection struct {
	headerEnd int
	end       int
	keys      []keyValue
}

type keyValue struct {
	key      string
	raw      unstable.Range
	value    unstable.Kind
	elements []unstable.Range
}

type configEditor struct {
	path       string
	mode       os.FileMode
	data       []byte
	cfg        File
	groupIndex int
	section    groupSection
}

func openConfigEditor(path, groupName string) (configEditor, error) {
	info, err := os.Stat(path)
	if err != nil {
		return configEditor{}, fmt.Errorf("stat config %s: %w", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return configEditor{}, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := decode(path, data)
	if err != nil {
		return configEditor{}, err
	}
	_, groupIndex, ok := FindGroup(cfg, groupName)
	if !ok {
		return configEditor{}, fmt.Errorf("unknown group %q", groupName)
	}
	section, err := findGroupSection(data, groupIndex)
	if err != nil {
		return configEditor{}, err
	}
	return configEditor{
		path:       path,
		mode:       info.Mode().Perm(),
		data:       data,
		cfg:        cfg,
		groupIndex: groupIndex,
		section:    section,
	}, nil
}

func (editor configEditor) commit(candidate []byte, expected File) error {
	decoded, err := decode(editor.path, candidate)
	if err != nil {
		return fmt.Errorf("validate edited config: %w", err)
	}
	if !reflect.DeepEqual(expected, decoded) {
		return fmt.Errorf("edited config does not match the expected configuration")
	}
	return writeAtomic(editor.path, candidate, editor.mode)
}

// AppendApp appends app to groupName without rewriting the rest of the TOML document.
func AppendApp(path, groupName string, app model.App) error {
	editor, err := openConfigEditor(path, groupName)
	if err != nil {
		return err
	}
	candidate, err := appendApp(editor.data, editor.section, app.Name)
	if err != nil {
		return err
	}
	expected := File{
		Lists:  editor.cfg.Lists,
		Groups: append([]Group(nil), editor.cfg.Groups...),
	}
	expected.Groups[editor.groupIndex].Apps = append(expected.Groups[editor.groupIndex].Apps, AppEntry{App: app})
	return editor.commit(candidate, expected)
}

// RemoveApp removes app from groupName without rewriting the rest of the TOML document.
func RemoveApp(path, groupName string, app model.App) error {
	editor, err := openConfigEditor(path, groupName)
	if err != nil {
		return err
	}
	group := editor.cfg.Groups[editor.groupIndex]
	appIndex := -1
	for index, entry := range group.Apps {
		if entry.App.Name == app.Name {
			appIndex = index
			break
		}
	}
	if appIndex < 0 {
		return fmt.Errorf("app %q does not exist in group %q", app.Name, groupName)
	}
	key, ok := appsKey(editor.section)
	if !ok || len(key.elements) != len(group.Apps) {
		return fmt.Errorf("cannot safely locate app %q in group %q", app.Name, groupName)
	}
	candidate, err := removeFromArray(editor.data, key, appIndex)
	if err != nil {
		return err
	}
	expected := File{
		Lists:  editor.cfg.Lists,
		Groups: append([]Group(nil), editor.cfg.Groups...),
	}
	apps := expected.Groups[editor.groupIndex].Apps
	remaining := make([]AppEntry, 0, len(apps)-1)
	remaining = append(remaining, apps[:appIndex]...)
	remaining = append(remaining, apps[appIndex+1:]...)
	expected.Groups[editor.groupIndex].Apps = remaining
	return editor.commit(candidate, expected)
}

func findGroupSection(data []byte, wanted int) (groupSection, error) {
	var parser unstable.Parser
	parser.KeepComments = true
	parser.Reset(data)

	var sections []groupSection
	current := -1
	for parser.NextExpression() {
		expr := parser.Expression()
		if isSectionHeader(expr) {
			current = advanceGroupSection(data, parser, expr, &sections, current)
			continue
		}
		if current >= 0 && expr.Kind == unstable.KeyValue {
			sections[current].keys = append(sections[current].keys, makeKeyValue(parser, expr))
		}
	}
	if err := parser.Error(); err != nil {
		return groupSection{}, fmt.Errorf("parse config for editing: %w", err)
	}
	if current >= 0 {
		sections[current].end = len(data)
	}
	if wanted >= len(sections) {
		return groupSection{}, fmt.Errorf("cannot locate source section for group %d", wanted)
	}
	return sections[wanted], nil
}

func isSectionHeader(expr *unstable.Node) bool {
	return expr.Kind == unstable.Table || expr.Kind == unstable.ArrayTable
}

func advanceGroupSection(data []byte, parser unstable.Parser, expr *unstable.Node, sections *[]groupSection, current int) int {
	if current >= 0 {
		(*sections)[current].end = expressionStart(data, expr)
	}
	if expr.Kind != unstable.ArrayTable || exactKey(parser, expr) != "groups" {
		return -1
	}
	*sections = append(*sections, groupSection{headerEnd: lineEnd(data, expressionStart(data, expr))})
	return len(*sections) - 1
}

func makeKeyValue(parser unstable.Parser, expr *unstable.Node) keyValue {
	key, _ := simpleKey(parser, expr)
	value := expr.Value()
	entry := keyValue{key: key, raw: expr.Raw, value: value.Kind}
	if value.Kind != unstable.Array {
		return entry
	}
	children := value.Children()
	for children.Next() {
		child := children.Node()
		if child.Kind != unstable.Comment {
			entry.elements = append(entry.elements, child.Raw)
		}
	}
	return entry
}

func exactKey(parser unstable.Parser, node *unstable.Node) string {
	key, ok := simpleKey(parser, node)
	if !ok {
		return ""
	}
	return key
}

func simpleKey(parser unstable.Parser, node *unstable.Node) (string, bool) {
	keys := node.Key()
	if !keys.Next() || !keys.IsLast() {
		return "", false
	}
	key := keys.Node()
	if key.Kind != unstable.Key {
		return "", false
	}
	return string(parser.Raw(key.Raw)), true
}

func expressionStart(data []byte, node *unstable.Node) int {
	if node.Kind == unstable.KeyValue {
		return int(node.Raw.Offset)
	}
	keys := node.Key()
	keys.Next()
	start := int(keys.Node().Raw.Offset)
	for start > 0 && data[start-1] != '\n' {
		start--
	}
	return start
}

func appendApp(data []byte, section groupSection, appName string) ([]byte, error) {
	key, ok := appsKey(section)
	if ok {
		return appendToArray(data, key, appName)
	}
	for _, key := range section.keys {
		if key.key == "apps" {
			return nil, fmt.Errorf("apps in selected group is not an array")
		}
	}

	insertAt := section.headerEnd
	for _, key := range section.keys {
		end := lineEnd(data, int(key.raw.Offset)+int(key.raw.Length))
		if end > insertAt {
			insertAt = end
		}
	}
	newline, err := newlineFor(data, section)
	if err != nil {
		return nil, err
	}
	insertion := []byte("apps = [" + strconv.Quote(appName) + "]" + newline)
	if insertAt > 0 && !isLineBoundary(data, insertAt) {
		insertion = append([]byte(newline), insertion...)
	}
	return splice(data, insertAt, insertion), nil
}

func appsKey(section groupSection) (keyValue, bool) {
	for _, key := range section.keys {
		if key.key != "apps" {
			continue
		}
		if key.value != unstable.Array {
			return keyValue{}, false
		}
		return key, true
	}
	return keyValue{}, false
}

func appendToArray(data []byte, key keyValue, appName string) ([]byte, error) {
	start, end, err := arrayBounds(data, key)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(data[start+1:end], []byte{'\n'}) {
		return appendInlineArray(data, key.elements, end, appName)
	}
	return appendMultilineArray(data, key.elements, start, end, appName)
}

func removeFromArray(data []byte, key keyValue, index int) ([]byte, error) {
	start, end, err := arrayBounds(data, key)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(key.elements) {
		return nil, fmt.Errorf("cannot establish app array element")
	}
	if bytes.Contains(data[start+1:end], []byte{'\n'}) {
		return removeMultilineArray(data, key.elements, index, end)
	}
	return removeInlineArray(data, key.elements, index, end)
}

func removeInlineArray(data []byte, values []unstable.Range, index, arrayEnd int) ([]byte, error) {
	valueStart := int(values[index].Offset)
	valueStop, err := valueEnd(data, values[index], arrayEnd)
	if err != nil {
		return nil, err
	}
	if len(values) == 1 {
		return cut(data, valueStart, valueStop), nil
	}
	if index < len(values)-1 {
		return cut(data, valueStart, int(values[index+1].Offset)), nil
	}
	previousEnd, err := valueEnd(data, values[index-1], valueStart)
	if err != nil {
		return nil, err
	}
	return cut(data, previousEnd, valueStop), nil
}

func removeMultilineArray(data []byte, values []unstable.Range, index, arrayEnd int) ([]byte, error) {
	valueStart := int(values[index].Offset)
	valueEnd, err := valueEnd(data, values[index], arrayEnd)
	if err != nil {
		return nil, err
	}
	lineEnd := valueEnd
	for lineEnd < len(data) && data[lineEnd] != '\n' {
		lineEnd++
	}
	cursor := valueEnd
	for cursor < lineEnd && (data[cursor] == ' ' || data[cursor] == '\t' || data[cursor] == '\r') {
		cursor++
	}
	if cursor == lineEnd {
		end := lineEnd
		if end < len(data) {
			end++
		}
		return cut(data, lineStart(data, valueStart), end), nil
	}
	if data[cursor] == ',' {
		cursor++
		for cursor < lineEnd && (data[cursor] == ' ' || data[cursor] == '\t') {
			cursor++
		}
		if cursor == lineEnd {
			end := lineEnd
			if end < len(data) {
				end++
			}
			return cut(data, lineStart(data, valueStart), end), nil
		}
		if data[cursor] == '#' {
			return cut(data, valueStart, cursor), nil
		}
	}
	if data[cursor] == '#' {
		return cut(data, valueStart, valueEnd), nil
	}
	return nil, fmt.Errorf("unsupported multiline apps layout")
}

func arrayBounds(data []byte, key keyValue) (start, end int, err error) {
	keyEnd := int(key.raw.Offset) + int(key.raw.Length)
	keyStart := int(key.raw.Offset)
	nameEnd := keyStart
	for nameEnd < keyEnd && data[nameEnd] != '=' {
		nameEnd++
	}
	if nameEnd == keyEnd {
		return 0, 0, fmt.Errorf("cannot locate apps assignment")
	}
	start = nameEnd + 1
	for start < keyEnd && (data[start] == ' ' || data[start] == '\t') {
		start++
	}
	if start >= keyEnd || data[start] != '[' || data[keyEnd-1] != ']' {
		return 0, 0, fmt.Errorf("cannot establish apps array boundary")
	}
	return start, keyEnd - 1, nil
}

func appendInlineArray(data []byte, values []unstable.Range, end int, appName string) ([]byte, error) {
	quoted := []byte(strconv.Quote(appName))
	if len(values) == 0 {
		return splice(data, end, quoted), nil
	}
	lastEnd, err := valueEnd(data, values[len(values)-1], end)
	if err != nil {
		return nil, err
	}
	if lastEnd > end {
		return nil, fmt.Errorf("cannot establish final apps value boundary")
	}
	trailing := data[lastEnd:end]
	if comma := bytes.IndexByte(trailing, ','); comma >= 0 {
		if len(bytes.TrimSpace(trailing[:comma])) != 0 {
			return nil, fmt.Errorf("unsupported inline apps layout")
		}
		return splice(data, end, quoted), nil
	}
	separator := inlineSeparator(data, values)
	return splice(data, lastEnd, append(separator, quoted...)), nil
}

func inlineSeparator(data []byte, values []unstable.Range) []byte {
	if len(values) < 2 {
		return []byte(", ")
	}
	previousEnd, err := valueEnd(data, values[len(values)-2], int(values[len(values)-1].Offset))
	if err != nil {
		return []byte(", ")
	}
	lastStart := int(values[len(values)-1].Offset)
	between := data[previousEnd:lastStart]
	if comma := bytes.IndexByte(between, ','); comma >= 0 {
		space := between[comma+1:]
		if len(bytes.TrimSpace(space)) == 0 {
			return append([]byte(","), space...)
		}
	}
	return []byte(", ")
}

func appendMultilineArray(data []byte, values []unstable.Range, start, end int, appName string) ([]byte, error) {
	newline := detectNewline(data[start:end])
	if newline == "" {
		return nil, fmt.Errorf("cannot establish multiline apps newline")
	}
	closeLineStart := lineStart(data, end)
	if len(bytes.TrimSpace(data[closeLineStart:end])) != 0 {
		return nil, fmt.Errorf("apps closing bracket must occupy its own line")
	}
	indent := multilineIndent(data, values, closeLineStart)
	if indent == "" {
		indent = lineIndent(data, closeLineStart) + "  "
	}
	trailingComma := false
	if len(values) > 0 {
		lastEnd, err := valueEnd(data, values[len(values)-1], closeLineStart)
		if err != nil {
			return nil, err
		}
		if lastEnd > closeLineStart {
			return nil, fmt.Errorf("cannot establish final multiline apps value boundary")
		}
		comma, err := finalValueComma(data, lastEnd)
		if err != nil {
			return nil, err
		}
		trailingComma = comma
		if !comma {
			data = splice(data, lastEnd, []byte(","))
			closeLineStart++
		}
	}
	value := indent + strconv.Quote(appName)
	if trailingComma {
		value += ","
	}
	return splice(data, closeLineStart, []byte(value+newline)), nil
}

func valueEnd(data []byte, raw unstable.Range, limit int) (int, error) {
	start := int(raw.Offset)
	if start >= limit {
		return 0, fmt.Errorf("cannot establish apps value boundary")
	}
	if data[start] != '{' && data[start] != '[' {
		end := start + int(raw.Length)
		if end > limit {
			return 0, fmt.Errorf("cannot establish apps value boundary")
		}
		return end, nil
	}

	depth := 0
	quote := byte(0)
	for index := start; index < limit; index++ {
		ch := data[index]
		if quote != 0 {
			if quote == '"' && ch == '\\' {
				index++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return index + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("cannot establish apps value boundary")
}

func finalValueComma(data []byte, valueEnd int) (bool, error) {
	lineEnd := valueEnd
	for lineEnd < len(data) && data[lineEnd] != '\n' {
		lineEnd++
	}
	between := bytes.TrimSpace(data[valueEnd:lineEnd])
	if len(between) == 0 {
		return false, nil
	}
	if bytes.Equal(between, []byte(",")) {
		return true, nil
	}
	if bytes.HasPrefix(between, []byte(",#")) || bytes.HasPrefix(between, []byte(", #")) {
		return true, nil
	}
	if bytes.HasPrefix(between, []byte("#")) {
		return false, nil
	}
	return false, fmt.Errorf("unsupported multiline apps layout")
}

func multilineIndent(data []byte, values []unstable.Range, closeLineStart int) string {
	if len(values) > 0 {
		return lineIndent(data, lineStart(data, int(values[len(values)-1].Offset)))
	}
	for offset := closeLineStart; offset > 0; {
		previous := lineStart(data, offset-1)
		line := data[previous:offset]
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("#")) {
			return lineIndent(data, previous)
		}
		if len(trimmed) > 0 {
			break
		}
		offset = previous
	}
	return ""
}

func newlineFor(data []byte, section groupSection) (string, error) {
	newline := detectNewline(data[section.headerEnd:section.end])
	if newline == "" {
		newline = detectNewline(data)
	}
	if newline == "" {
		return "", fmt.Errorf("cannot establish config newline sequence")
	}
	return newline, nil
}

func detectNewline(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	if bytes.Contains(data, []byte("\n")) {
		return "\n"
	}
	return ""
}

func lineStart(data []byte, offset int) int {
	for offset > 0 && data[offset-1] != '\n' {
		offset--
	}
	return offset
}

func lineEnd(data []byte, offset int) int {
	for offset < len(data) && data[offset] != '\n' {
		offset++
	}
	if offset < len(data) {
		offset++
	}
	return offset
}

func lineIndent(data []byte, offset int) string {
	end := offset
	for end < len(data) && (data[end] == ' ' || data[end] == '\t') {
		end++
	}
	return string(data[offset:end])
}

func isLineBoundary(data []byte, offset int) bool {
	return offset == 0 || offset == len(data) || data[offset-1] == '\n'
}

func splice(data []byte, offset int, insertion []byte) []byte {
	result := make([]byte, 0, len(data)+len(insertion))
	result = append(result, data[:offset]...)
	result = append(result, insertion...)
	result = append(result, data[offset:]...)
	return result
}

func cut(data []byte, start, end int) []byte {
	result := make([]byte, 0, len(data)-(end-start))
	result = append(result, data[:start]...)
	result = append(result, data[end:]...)
	return result
}

func writeAtomic(path string, data []byte, mode os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bmx-config-*.toml")
	if err != nil {
		return fmt.Errorf("create config temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if err = tmp.Chmod(mode); err != nil {
		return fmt.Errorf("set config temp file mode: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write config temp file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync config temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close config temp file: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config file: %w", err)
	}
	return nil
}

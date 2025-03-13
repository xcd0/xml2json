package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/pkg/errors"
)

// グローバル変数。
var (
	reMixedContentIndex = regexp.MustCompile(`^\$(\d+)$`)
	alwaysArrayElements = map[string]bool{
		"col":   true,
		"row":   true,
		"table": true,
	}
	ToXML = false // JSONからXMLへの変換モード
)

func main() {
	var inputString []byte
	var output io.Writer
	var err error

	if len(os.Args) == 2 {
		args.InputFile = os.Args[1]

		var input *os.File
		input, err = os.Open(args.InputFile)
		if err != nil {
			panic(errors.Errorf("入力ファイルを開けません: %v", err))
		}
		defer input.Close()

		inputString, err = io.ReadAll(input)
		if err != nil {
			panic(errors.Errorf("標準入力の読み込みに失敗しました: %v", err))
		}

		if e := strings.ToLower(filepath.Ext(args.InputFile)); e == ".xml" {
			args.OutputFile = args.InputFile + ".json"
		} else if e == ".json" {
			args.OutputFile = args.InputFile + ".xml"
			ToXML = true
		}
		if args.OutputFile != "" {
			file, err := os.Create(args.OutputFile)
			if err != nil {
				panic(errors.Errorf("出力ファイルを作成できません: %v", err))
			}
			defer file.Close()
			output = file
		} else {
			output = os.Stdout
		}
	} else {
		ParseArgs()

		ToXML = args.ToXML
		if args.Debug {
			fmt.Fprintf(os.Stderr, "入力ファイル: %s\n", args.InputFile)
			fmt.Fprintf(os.Stderr, "出力ファイル: %s\n", args.OutputFile)
			fmt.Fprintf(os.Stderr, "変換モード: %s\n", map[bool]string{false: "XMLからJSON", true: "JSONからXML"}[args.ToXML])
		}

		if len(args.InputFile) == 0 && len(os.Args) == 1 {
			// 標準入力から読み取り、標準出力に出力する。
			inputString, err = io.ReadAll(os.Stdin)
			if err != nil {
				panic(errors.Errorf("標準入力の読み込みに失敗しました: %v", err))
			}
		} else {
			var input *os.File
			input, err = os.Open(args.InputFile)
			if err != nil {
				panic(errors.Errorf("入力ファイルを開けません: %v", err))
			}
			defer input.Close()

			inputString, err = io.ReadAll(input)
			if err != nil {
				panic(errors.Errorf("入力ファイルの読み込みに失敗しました: %v", err))
			}
		}
		if args.OutputFile != "" {
			file, err := os.Create(args.OutputFile)
			if err != nil {
				panic(errors.Errorf("出力ファイルを作成できません: %v", err))
			}
			defer file.Close()
			output = file
		} else {
			output = os.Stdout
		}
	}

	if !ToXML {
		ConvertXMLToJSON(inputString, output)
	} else {
		ConvertJSONToXML(inputString, output)
	}
}

// ---------------------------------------------------------------------
// XMLからJSONへの変換処理
// ---------------------------------------------------------------------

func ConvertXMLToJSON(inputString []byte, output io.Writer) {
	decoder := xml.NewDecoder(bytes.NewReader(inputString))
	root := make(map[string]interface{})
	orderMap := make(map[string][]string)
	comments := []string{}
	processingInstructions := []map[string]string{}
	var doctype string

	elementStack := []map[string]interface{}{}
	nameStack := []string{}
	currentElement := root

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(errors.Errorf("XMLのパースに失敗しました: %v", err))
		}

		switch t := token.(type) {
		case xml.StartElement:
			element := make(map[string]interface{})
			// 属性の並び順を記録する。
			var attrOrder []string
			for _, attr := range t.Attr {
				var attrName string
				if attr.Name.Space != "" {
					attrName = "@" + attr.Name.Space + ":" + attr.Name.Local
				} else {
					attrName = "@" + attr.Name.Local
				}
				attrOrder = append(attrOrder, attrName)

				if attr.Name.Space == "xmlns" {
					if element["@xmlns"] == nil {
						element["@xmlns"] = make(map[string]interface{})
					}
					namespaces := element["@xmlns"].(map[string]interface{})
					if attr.Name.Local == "" {
						namespaces["$"] = attr.Value
					} else {
						namespaces[attr.Name.Local] = attr.Value
					}
				} else {
					element[attrName] = attr.Value
				}
			}
			if len(attrOrder) > 0 {
				// $attrOrder をそのまま保存する。
				element["$attrOrder"] = attrOrder
			}

			elementName := t.Name.Local
			if t.Name.Space != "" {
				elementName = t.Name.Space + ":" + elementName
			}
			nameStack = append(nameStack, elementName)
			if len(nameStack) > 1 {
				parentPath := strings.Join(nameStack[:len(nameStack)-1], "/")
				if _, exists := orderMap[parentPath]; !exists {
					orderMap[parentPath] = []string{}
				}
				found := false
				for _, name := range orderMap[parentPath] {
					if name == elementName {
						found = true
						break
					}
				}
				if !found {
					orderMap[parentPath] = append(orderMap[parentPath], elementName)
				}
			}

			if existingElement, ok := currentElement[elementName]; ok {
				if array, ok := existingElement.([]interface{}); ok {
					currentElement[elementName] = append(array, element)
				} else {
					currentElement[elementName] = []interface{}{existingElement, element}
				}
			} else {
				if alwaysArrayElements[elementName] {
					currentElement[elementName] = []interface{}{element}
				} else {
					currentElement[elementName] = element
				}
			}

			elementStack = append(elementStack, currentElement)
			if array, ok := currentElement[elementName].([]interface{}); ok {
				currentElement = array[len(array)-1].(map[string]interface{})
			} else {
				currentElement = currentElement[elementName].(map[string]interface{})
			}

		case xml.EndElement:
			if len(elementStack) > 0 {
				currentElement = elementStack[len(elementStack)-1]
				elementStack = elementStack[:len(elementStack)-1]
				if len(nameStack) > 0 {
					nameStack = nameStack[:len(nameStack)-1]
				}
			}

		case xml.CharData:
			text := string(t)
			if strings.TrimSpace(text) != "" {
				currentElement["$"] = text
			}

		case xml.Comment:
			commentText := string(t)
			comments = append(comments, commentText)
			if len(elementStack) == 0 {
				root["$comment"] = comments
			}

		case xml.ProcInst:
			pi := map[string]string{
				"target": t.Target,
				"data":   string(t.Inst),
			}
			processingInstructions = append(processingInstructions, pi)
			if len(elementStack) == 0 {
				root["$pi"] = processingInstructions
			}

		case xml.Directive:
			directiveText := string(t)
			if strings.HasPrefix(strings.TrimSpace(directiveText), "DOCTYPE") {
				doctype = "<!" + string(t) + ">"
				if len(elementStack) == 0 {
					root["$doctype"] = doctype
				}
			}
		}
	}

	if len(comments) > 0 && root["$comment"] == nil {
		root["$comment"] = comments
	}
	if len(processingInstructions) > 0 && root["$pi"] == nil {
		root["$pi"] = processingInstructions
	}
	if doctype != "" && root["$doctype"] == nil {
		root["$doctype"] = doctype
	}

	root["$orderMap"] = orderMap

	var jsonData []byte
	var err error
	if args.Minify {
		jsonData, err = json.Marshal(root)
	} else {
		jsonData, err = json.MarshalIndent(root, "", "\t")
	}
	if err != nil {
		panic(errors.Errorf("JSONへの変換に失敗しました: %v", err))
	}

	// 出力直前に改行コードをCRLFに統一する
	normalized := normalizeNewlinesToCRLF(string(jsonData))
	_, err = output.Write([]byte(normalized))
	if err != nil {
		panic(errors.Errorf("JSONデータの書き込みに失敗しました: %v", err))
	}
}

// ---------------------------------------------------------------------
// JSONからXMLへの変換処理
// ---------------------------------------------------------------------

func ConvertJSONToXML(inputString []byte, output io.Writer) {
	var root map[string]interface{}
	err := json.Unmarshal(inputString, &root)
	if err != nil {
		panic(errors.Errorf("JSONのパースに失敗しました: %v", err))
	}

	var orderMap map[string][]string
	if orderData, ok := root["$orderMap"]; ok {
		orderMap = make(map[string][]string)
		if orderMapData, ok := orderData.(map[string]interface{}); ok {
			for path, value := range orderMapData {
				if childArr, ok := value.([]interface{}); ok {
					orderMap[path] = make([]string, len(childArr))
					for i, v := range childArr {
						if s, ok := v.(string); ok {
							orderMap[path][i] = s
						}
					}
				}
			}
		}
		delete(root, "$orderMap")
	}

	var buffer bytes.Buffer
	declarations := make(map[string]string)
	processingInstructions := []map[string]string{}
	var doctype string
	var comments []string

	if piValue, ok := root["$pi"]; ok {
		if piArray, ok := piValue.([]interface{}); ok {
			for _, piItem := range piArray {
				if pi, ok := piItem.(map[string]interface{}); ok {
					target, _ := pi["target"].(string)
					data, _ := pi["data"].(string)
					if target == "xml" {
						declarations["xml"] = data
					} else {
						processingInstructions = append(processingInstructions, map[string]string{
							"target": target,
							"data":   data,
						})
					}
				}
			}
		}
		delete(root, "$pi")
	}

	if doctypeValue, ok := root["$doctype"]; ok {
		if doctypeStr, ok := doctypeValue.(string); ok {
			doctype = doctypeStr
		}
		delete(root, "$doctype")
	}

	if commentValue, ok := root["$comment"]; ok {
		if commentArray, ok := commentValue.([]interface{}); ok {
			for _, commentItem := range commentArray {
				if comment, ok := commentItem.(string); ok {
					comments = append(comments, comment)
				}
			}
		}
		delete(root, "$comment")
	}

	if xmlDecl, ok := declarations["xml"]; ok {
		buffer.WriteString("<?xml " + xmlDecl + "?>\n")
	} else {
		buffer.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	}

	for _, pi := range processingInstructions {
		buffer.WriteString("<?")
		buffer.WriteString(pi["target"])
		buffer.WriteString(" ")
		buffer.WriteString(pi["data"])
		buffer.WriteString("?>\n")
	}

	if doctype != "" {
		buffer.WriteString(doctype + "\n")
	}

	for _, comment := range comments {
		buffer.WriteString("<!--")
		buffer.WriteString(comment)
		buffer.WriteString("-->\n")
	}

	// 初期の名前空間コンテキストは空で開始
	for elementName, elementValue := range root {
		if strings.HasPrefix(elementName, "$") {
			continue
		}
		writeXMLElement(&buffer, elementName, elementValue, 0, orderMap, make(map[string]string))
	}

	result := buffer.Bytes()
	if !args.Minify {
		result = []byte(strings.TrimLeft(xmlfmt.FormatXML(string(result), "", "\t"), "\r\n"))
	}

	// 出力直前に改行コードをCRLFに統一する
	normalized := normalizeNewlinesToCRLF(string(result))
	_, err = output.Write([]byte(normalized))
	if err != nil {
		panic(errors.Errorf("XMLデータの書き込みに失敗しました: %v", err))
	}
}

// writeXMLElement は名前空間コンテキストを受け取り、属性の並び順 ($attrOrder) を考慮して出力する。
func writeXMLElement(buffer *bytes.Buffer, name string, value interface{}, indent int, orderMap map[string][]string, nsContext map[string]string) {
	// 配列の場合、各要素を個別に処理。
	if arr, ok := value.([]interface{}); ok {
		for _, item := range arr {
			writeXMLElement(buffer, name, item, indent, orderMap, nsContext)
		}
		return
	}

	buffer.WriteString("<")
	buffer.WriteString(name)

	// 名前空間コンテキストのローカルコピーを作成。
	localNS := make(map[string]string)
	for k, v := range nsContext {
		localNS[k] = v
	}

	// 属性を、$attrOrder があればその順序で出力する。
	if element, ok := value.(map[string]interface{}); ok {
		var rawAttrKeys []string
		attrMap := make(map[string]interface{})
		for k, v := range element {
			if strings.HasPrefix(k, "@") {
				rawAttrKeys = append(rawAttrKeys, k)
				attrMap[k] = v
			}
		}
		var outputAttrKeys []string
		if orderVal, ok := element["$attrOrder"]; ok {
			var orderSlice []string
			if arr, ok := orderVal.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						orderSlice = append(orderSlice, s)
					}
				}
			} else if arr, ok := orderVal.([]string); ok {
				orderSlice = arr
			}
			// まず、orderSlice に記載されたキー順に出力する。
			for _, key := range orderSlice {
				if _, exists := attrMap[key]; exists {
					outputAttrKeys = append(outputAttrKeys, key)
				}
			}
			// orderSlice にない残りの属性キーを追加（昇順）。
			for _, key := range rawAttrKeys {
				found := false
				for _, oKey := range orderSlice {
					if key == oKey {
						found = true
						break
					}
				}
				if !found {
					outputAttrKeys = append(outputAttrKeys, key)
				}
			}
			delete(element, "$attrOrder")
		} else {
			outputAttrKeys = rawAttrKeys
			sort.Strings(outputAttrKeys)
		}

		// xmlns 属性は別途集める。
		var xmlnsAttrs []string
		for _, attrKey := range outputAttrKeys {
			v := attrMap[attrKey]
			if attrKey == "@xmlns" {
				if nsMap, ok := v.(map[string]interface{}); ok {
					for prefix, uri := range nsMap {
						xmlnsAttrs = append(xmlnsAttrs, fmt.Sprintf(" xmlns:%s=\"%s\"", prefix, uri))
						localNS[prefix] = fmt.Sprintf("%s", uri)
					}
				}
			} else {
				rawName := attrKey[1:]
				// 名前空間付き属性の場合、ローカルコンテキストからプレフィックスを再設定。
				if idx := strings.LastIndex(rawName, ":"); idx != -1 {
					nsURI := rawName[:idx]
					localName := rawName[idx+1:]
					prefix := ""
					for p, uri := range localNS {
						if uri == nsURI {
							prefix = p
							break
						}
					}
					if prefix != "" {
						rawName = prefix + ":" + localName
					}
				}
				buffer.WriteString(" ")
				buffer.WriteString(rawName)
				buffer.WriteString("=\"")
				buffer.WriteString(escapeXMLAttr(fmt.Sprintf("%v", v)))
				buffer.WriteString("\"")
			}
		}
		// 出力する xmlns 宣言。
		for _, xmlns := range xmlnsAttrs {
			buffer.WriteString(xmlns)
		}

		// 子要素と内容の有無をチェック。
		hasContent := false
		for key := range element {
			if !strings.HasPrefix(key, "@") {
				hasContent = true
				break
			}
		}
		if !hasContent {
			buffer.WriteString("/>")
			return
		}
		buffer.WriteString(">")

		// テキスト内容の処理。
		if textValue, ok := element["$"]; ok {
			buffer.WriteString(preserveXMLEntities(fmt.Sprintf("%v", textValue)))
		}
		if cdataValue, ok := element["$cdata"]; ok {
			buffer.WriteString("<![CDATA[")
			buffer.WriteString(fmt.Sprintf("%v", cdataValue))
			buffer.WriteString("]]>")
		}
		if rawValue, ok := element["$raw"]; ok {
			buffer.WriteString(fmt.Sprintf("%v", rawValue))
		}

		// table 要素特有の処理。
		if name == "table" {
			if colsValue, ok := element["col"]; ok {
				if colArray, ok := colsValue.([]interface{}); ok {
					for _, colItem := range colArray {
						if colMap, ok := colItem.(map[string]interface{}); ok {
							buffer.WriteString("\n\t<col")
							// col の属性も順序を維持して出力。
							var colRawAttrKeys []string
							colAttrMap := make(map[string]interface{})
							for k, v := range colMap {
								if strings.HasPrefix(k, "@") {
									colRawAttrKeys = append(colRawAttrKeys, k)
									colAttrMap[k] = v
								}
							}
							var colOutputAttrKeys []string
							if orderVal, ok := colMap["$attrOrder"]; ok {
								var orderSlice []string
								if arr, ok := orderVal.([]interface{}); ok {
									for _, v := range arr {
										if s, ok := v.(string); ok {
											orderSlice = append(orderSlice, s)
										}
									}
								} else if arr, ok := orderVal.([]string); ok {
									orderSlice = arr
								}
								for _, key := range orderSlice {
									if _, exists := colAttrMap[key]; exists {
										colOutputAttrKeys = append(colOutputAttrKeys, key)
									}
								}
								for _, key := range colRawAttrKeys {
									found := false
									for _, oKey := range orderSlice {
										if key == oKey {
											found = true
											break
										}
									}
									if !found {
										colOutputAttrKeys = append(colOutputAttrKeys, key)
									}
								}
								delete(colMap, "$attrOrder")
							} else {
								colOutputAttrKeys = colRawAttrKeys
								sort.Strings(colOutputAttrKeys)
							}
							for _, attrKey := range colOutputAttrKeys {
								v := colAttrMap[attrKey]
								rawName := attrKey[1:]
								buffer.WriteString(" ")
								buffer.WriteString(rawName)
								buffer.WriteString("=\"")
								buffer.WriteString(escapeXMLAttr(fmt.Sprintf("%v", v)))
								buffer.WriteString("\"")
							}
							if textContent, ok := colMap["$"]; ok {
								buffer.WriteString(">")
								buffer.WriteString(preserveXMLEntities(fmt.Sprintf("%v", textContent)))
								buffer.WriteString("</col>")
							} else {
								buffer.WriteString("/>")
							}
						}
					}
				}
			}
			if rowsValue, ok := element["row"]; ok {
				if rowArray, ok := rowsValue.([]interface{}); ok {
					for _, rowItem := range rowArray {
						buffer.WriteString("\n\t<row>")
						if rowMap, ok := rowItem.(map[string]interface{}); ok {
							if tdValue, ok := rowMap["td"]; ok {
								// td 要素の処理も同様に。
								if tdArray, ok := tdValue.([]interface{}); ok {
									for _, tdItem := range tdArray {
										buffer.WriteString("\n\t\t<td")
										if tdMap, ok := tdItem.(map[string]interface{}); ok {
											var tdRawAttrKeys []string
											tdAttrMap := make(map[string]interface{})
											for k, v := range tdMap {
												if strings.HasPrefix(k, "@") {
													tdRawAttrKeys = append(tdRawAttrKeys, k)
													tdAttrMap[k] = v
												}
											}
											var tdOutputAttrKeys []string
											if orderVal, ok := tdMap["$attrOrder"]; ok {
												var orderSlice []string
												if arr, ok := orderVal.([]interface{}); ok {
													for _, v := range arr {
														if s, ok := v.(string); ok {
															orderSlice = append(orderSlice, s)
														}
													}
												} else if arr, ok := orderVal.([]string); ok {
													orderSlice = arr
												}
												for _, key := range orderSlice {
													if _, exists := tdAttrMap[key]; exists {
														tdOutputAttrKeys = append(tdOutputAttrKeys, key)
													}
												}
												for _, key := range tdRawAttrKeys {
													found := false
													for _, oKey := range orderSlice {
														if key == oKey {
															found = true
															break
														}
													}
													if !found {
														tdOutputAttrKeys = append(tdOutputAttrKeys, key)
													}
												}
												delete(tdMap, "$attrOrder")
											} else {
												tdOutputAttrKeys = tdRawAttrKeys
												sort.Strings(tdOutputAttrKeys)
											}
											for _, attrKey := range tdOutputAttrKeys {
												v := tdAttrMap[attrKey]
												rawName := attrKey[1:]
												if idx := strings.LastIndex(rawName, ":"); idx != -1 {
													nsURI := rawName[:idx]
													localName := rawName[idx+1:]
													prefix := ""
													for p, uri := range localNS {
														if uri == nsURI {
															prefix = p
															break
														}
													}
													if prefix != "" {
														rawName = prefix + ":" + localName
													}
												}
												buffer.WriteString(" ")
												buffer.WriteString(rawName)
												buffer.WriteString("=\"")
												buffer.WriteString(escapeXMLAttr(fmt.Sprintf("%v", v)))
												buffer.WriteString("\"")
											}
											buffer.WriteString(">")
											if textContent, ok := tdMap["$"]; ok {
												buffer.WriteString(preserveXMLEntities(fmt.Sprintf("%v", textContent)))
											}
											buffer.WriteString("</td>")
										}
									}
								} else if tdMap, ok := tdValue.(map[string]interface{}); ok {
									buffer.WriteString("\n\t\t<td")
									var tdRawAttrKeys []string
									tdAttrMap := make(map[string]interface{})
									for k, v := range tdMap {
										if strings.HasPrefix(k, "@") {
											tdRawAttrKeys = append(tdRawAttrKeys, k)
											tdAttrMap[k] = v
										}
									}
									var tdOutputAttrKeys []string
									if orderVal, ok := tdMap["$attrOrder"]; ok {
										var orderSlice []string
										if arr, ok := orderVal.([]interface{}); ok {
											for _, v := range arr {
												if s, ok := v.(string); ok {
													orderSlice = append(orderSlice, s)
												}
											}
										} else if arr, ok := orderVal.([]string); ok {
											orderSlice = arr
										}
										for _, key := range orderSlice {
											if _, exists := tdAttrMap[key]; exists {
												tdOutputAttrKeys = append(tdOutputAttrKeys, key)
											}
										}
										for _, key := range tdRawAttrKeys {
											found := false
											for _, oKey := range orderSlice {
												if key == oKey {
													found = true
													break
												}
											}
											if !found {
												tdOutputAttrKeys = append(tdOutputAttrKeys, key)
											}
										}
										delete(tdMap, "$attrOrder")
									} else {
										tdOutputAttrKeys = tdRawAttrKeys
										sort.Strings(tdOutputAttrKeys)
									}
									for _, attrKey := range tdOutputAttrKeys {
										v := tdAttrMap[attrKey]
										rawName := attrKey[1:]
										if idx := strings.LastIndex(rawName, ":"); idx != -1 {
											nsURI := rawName[:idx]
											localName := rawName[idx+1:]
											prefix := ""
											for p, uri := range localNS {
												if uri == nsURI {
													prefix = p
													break
												}
											}
											if prefix != "" {
												rawName = prefix + ":" + localName
											}
										}
										buffer.WriteString(" ")
										buffer.WriteString(rawName)
										buffer.WriteString("=\"")
										buffer.WriteString(escapeXMLAttr(fmt.Sprintf("%v", v)))
										buffer.WriteString("\"")
									}
									buffer.WriteString(">")
									if textContent, ok := tdMap["$"]; ok {
										buffer.WriteString(preserveXMLEntities(fmt.Sprintf("%v", textContent)))
									}
									buffer.WriteString("</td>")
								}
							}
						}
						buffer.WriteString("\n\t</row>")
					}
				}
			}
		} else {
			var childKeys []string
			for key := range element {
				if !strings.HasPrefix(key, "@") && !strings.HasPrefix(key, "$") && key != "col" && key != "row" {
					childKeys = append(childKeys, key)
				}
			}
			if len(childKeys) > 0 {
				sort.Strings(childKeys)
				if name == "summary" && orderMap != nil {
					if order, ok := orderMap["msi/summary"]; ok {
						sort.SliceStable(childKeys, func(i, j int) bool {
							keyI := childKeys[i]
							keyJ := childKeys[j]
							indexI, indexJ := -1, -1
							for idx, key := range order {
								if key == keyI {
									indexI = idx
								}
								if key == keyJ {
									indexJ = idx
								}
							}
							if indexI >= 0 && indexJ >= 0 {
								return indexI < indexJ
							}
							if indexI >= 0 {
								return true
							}
							if indexJ >= 0 {
								return false
							}
							return keyI < keyJ
						})
					}
				}
			}
			for _, key := range childKeys {
				childValue := element[key]
				writeXMLElement(buffer, key, childValue, indent+1, orderMap, localNS)
			}
		}

		buffer.WriteString("</")
		buffer.WriteString(name)
		buffer.WriteString(">")
	} else {
		buffer.WriteString(">")
		if value != nil {
			buffer.WriteString(preserveXMLEntities(fmt.Sprintf("%v", value)))
		}
		buffer.WriteString("</")
		buffer.WriteString(name)
		buffer.WriteString(">")
	}
}

func escapeXMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func escapeXMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func preserveXMLEntities(s string) string {
	entityPattern := regexp.MustCompile(`&(amp|lt|gt|quot|apos|#\d+);`)
	placeholderStr := "PLACEHOLDER_"
	count := 0
	placeholders := make(map[string]string)
	result := entityPattern.ReplaceAllStringFunc(s, func(entity string) string {
		key := fmt.Sprintf("%s%d", placeholderStr, count)
		placeholders[key] = entity
		count++
		return key
	})
	result = escapeXMLText(result)
	for key, entity := range placeholders {
		result = strings.Replace(result, key, entity, 1)
	}
	return result
}

// 改行コードをCRLFに統一する関数
func normalizeNewlinesToCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

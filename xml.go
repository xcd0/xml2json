package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

//xml_declarationをキーとして xml宣言 を文字列として出力、
//xml_document_type_definitionをキーとして xml文書型定義 を文字列として出力、
//xml_dataをキーとしてxmlのデータを出力する。

// XMLをJSON形式に変換するための構造体
type XMLToJSON struct {
	XMLDeclaration            []string               `json:"xml_declaration"`
	XMLDocumentTypeDefinition string                 `json:"xml_document_type_definition"`
	XMLData                   map[string]interface{} `json:"xml_data"`
}

func XmlJsonConverter(r io.Reader, toJsonFromXml bool) string {
	if toJsonFromXml {
		// XMLデータをJSONに変換
		//xmlData := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root SYSTEM "example.dtd"><root><element attribute="value">Text Content</element><emptyElement/></root>`
		//xmlReader := bytes.NewReader([]byte(xmlData))
		//jsonResult, err := convertXMLToJSON(xmlReader)
		jsonResult, err := convertXMLToJSON(r)
		if err != nil {
			panic(errors.Errorf("XML to JSON conversion error: %v", err))
		}
		return jsonResult
	} else {
		// JSONデータをXMLに変換
		//jsonReader := bytes.NewReader([]byte(jsonResult))
		//xmlResult, err := convertJSONToXML(jsonReader)
		xmlResult, err := convertJSONToXML(r)
		if err != nil {
			panic(errors.Errorf("JSON to XML conversion error: %v", err))
		}
		return xmlResult
	}
}

// mapをXMLに変換する再帰関数
func mapToXML(parent *xml.Encoder, startElement xml.StartElement, data map[string]interface{}) error {
	err := parent.EncodeToken(startElement)
	if err != nil {
		return err
	}

	for key, value := range data {
		switch v := value.(type) {
		case string:
			if key == "#text" {
				err := parent.EncodeToken(xml.CharData(v))
				if err != nil {
					return err
				}
			}
		case map[string]interface{}:
			err := mapToXML(parent, xml.StartElement{Name: xml.Name{Local: key}}, v)
			if err != nil {
				return err
			}
			err = parent.EncodeToken(xml.EndElement{Name: xml.Name{Local: key}})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// XMLをJSONに変換する関数
func convertXMLToJSON(reader io.Reader) (string, error) {
	decoder := xml.NewDecoder(reader)
	var result XMLToJSON
	rootMap := make(map[string]interface{})
	var currentMap map[string]interface{} = rootMap
	var stack []map[string]interface{}
	var keyStack []string

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		switch se := t.(type) {
		case xml.ProcInst:
			// 複数のXML宣言を順序を保ってスライスに追加
			result.XMLDeclaration = append(result.XMLDeclaration, fmt.Sprintf("<?%s %s?>", se.Target, string(se.Inst)))
		case xml.Directive:
			if strings.HasPrefix(string(se), "DOCTYPE") {
				result.XMLDocumentTypeDefinition = fmt.Sprintf("<!%s>", string(se))
			}
		case xml.StartElement:
			elementMap := make(map[string]interface{})
			for _, attr := range se.Attr {
				elementMap[attr.Name.Local] = attr.Value
			}
			if len(currentMap) == 0 {
				currentMap[se.Name.Local] = elementMap
			} else {
				// スタックに現在のマップを保持
				stack = append(stack, currentMap)
				keyStack = append(keyStack, se.Name.Local)
				currentMap[se.Name.Local] = elementMap
			}
			currentMap = elementMap

		case xml.EndElement:
			if len(stack) > 0 {
				// スタックからマップをポップ
				currentMap = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				keyStack = keyStack[:len(keyStack)-1]
			}

		case xml.CharData:
			content := string(bytes.TrimSpace(se))
			if len(content) > 0 {
				currentMap["#text"] = content
			}
		}
	}

	result.XMLData = rootMap

	// JSONにエンコード
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// JSONをXMLに戻す関数
func convertJSONToXML(reader io.Reader) (string, error) {
	var parsedJSON XMLToJSON

	// io.ReaderからJSONデータを読み込んでパース
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&parsedJSON)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer

	// 複数のXML宣言を順序を保って追加
	for _, declaration := range parsedJSON.XMLDeclaration {
		buffer.WriteString(declaration + "\n")
	}

	// DTDを追加
	if parsedJSON.XMLDocumentTypeDefinition != "" {
		buffer.WriteString(parsedJSON.XMLDocumentTypeDefinition + "\n")
	}

	// XMLデータをエンコード
	encoder := xml.NewEncoder(&buffer)
	encoder.Indent("", "\t") // インデント設定を追加
	for key, value := range parsedJSON.XMLData {
		startElement := xml.StartElement{Name: xml.Name{Local: key}}
		err := mapToXML(encoder, startElement, value.(map[string]interface{}))
		if err != nil {
			return "", err
		}
		err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: key}})
		if err != nil {
			return "", err
		}
	}
	encoder.Flush()

	// フォーマットされたXMLを文字列として返す
	return strings.TrimSpace(buffer.String()), nil
}

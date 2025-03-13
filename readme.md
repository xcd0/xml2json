# XML-JSON相互変換ツール仕様

## 概要
XMLとJSON間で情報の損失なく双方向変換を行うツールです。
BadgerFish拡張仕様に基づき、XML文書の全ての情報（要素、属性、テキスト内容、処理命令など）をJSON形式で表現し、元のXMLに完全に復元できます。
## 主要機能
1. **XMLからJSONへの変換**
   - 要素の階層構造を正確に保持
   - 属性は`@`プレフィックス付きのプロパティとして表現
   - テキスト内容は`$`プロパティに格納
   - 順序情報を`$orderMap`として保存
2. **JSONからXMLへの変換**
   - JSON形式から元のXML構造を正確に復元
   - 要素の順序を元のXMLと同様に保持
   - 特殊要素（table, col, row, td）の適切な処理
3. **特殊コンテンツの処理**
   - XML宣言、DOCTYPE、処理命令、コメントの保持
   - 名前空間の適切な処理
   - 混合コンテンツの保持
## コマンドライン引数
- `-i, --input-file`: 入力ファイルパス
- `-o, --output-file`: 出力ファイルパス（省略時は標準出力）
- `-j, --to-json`: XMLからJSONへの変換モード（デフォルト）
- `-x, --to-xml`: JSONからXMLへの変換モード
- `-m, --minify`: 整形出力を無効にする
- `-d, --debug`: デバッグ出力を有効にする
## 使用例
```bash
# XMLからJSONへの変換
./xml2json -i sample.xml -o sample.xml.json
# JSONからXMLへの変換
./xml2json --to-xml -i sample.xml.json -o sample.xml.json.xml
```

`sample.xml`と`sample.xml.json.xml`が一致する。


## 変換ルール
- 要素名はJSONオブジェクトのプロパティ名になる
- 属性は`@`プレフィックス付きのプロパティとして表現
- テキスト内容は`$`プロパティに格納
- 同名の複数要素は配列として表現
- 順序情報は`$orderMap`に保存
- 特殊命令は`$doctype`, `$pi`, `$comment`などに格納
この実装により、複雑なXML文書でも情報損失なく変換・復元が可能になります。


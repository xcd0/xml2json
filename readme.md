# xml2json

なんかいい感じのxml → jsonn変換コマンドがなかったから作った。
引数にxmlのファイルを与える。 複数個可。

```
xml2json <xml files...>
```
元の名前に`.json`をつけた名前で保存する。

```
$ xml2json tmp.xml 
$ ls
tmp.xml  tmp.xml.json

```

## install

```
go install github.com/xcd0/xml2json@latest
```

ビルド環境がない場合は https://github.com/Songmu/ghg を使えばgithubのリリースからとれる。
```
ghg get xcd0/xml2json
```

## 注意

メンテナンスされていないライブラリ
https://github.com/basgys/goxml2json
を使用している。  
現状問題がないが、問題が起きたらどうにかする\_(:3 」∠ )_\i



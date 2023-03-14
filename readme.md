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

## 注意

メンテナンスされていないライブラリを使用している。
https://github.com/basgys/goxml2json
現状問題がないが、問題が起きたらどうにかする\_(:3 」∠ )_\i



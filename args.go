package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/pkg/errors"
)

// コマンドライン引数の構造体。
type Args struct {
	InputFile  string `arg:"-i,--input-file"  help:"入力ファイルパス"                      placeholder:"SRC"`
	OutputFile string `arg:"-o,--output-file" help:"出力ファイルパス（省略時は標準出力）"  placeholder:"DST"`
	ToXML      bool   `arg:"-x,--to-xml"      help:"JSONからXMLへの変換モード"`
	Minify     bool   `arg:"-m,--minify"      help:"整形出力を無効にする"`
	Debug      bool   `arg:"-d,--debug"       help:"デバッグ出力を有効にする"`
	ExportCode string `arg:"--code"           help:"バイナリに埋め込まれているソースコードを指定パスに出力する。"  placeholder:"DST"`
}

func (Args) Version() string {
	return GetVersion()
}

// グローバル変数。
var (
	args   Args
	parser *arg.Parser // ShowHelp() で使う
)

// コマンドライン引数の解析。
func ParseArgs() {
	var err error
	parser, err = arg.NewParser(arg.Config{Program: GetFileNameWithoutExt(os.Args[0]), IgnoreEnv: false}, &args)
	if err != nil {
		ShowHelp(fmt.Sprintf("%v", errors.Errorf("%v", err)))
		os.Exit(1)
	}

	err = parser.Parse(os.Args[1:])
	if err != nil {
		if err.Error() == "help requested by user" {
			ShowHelp("")
			os.Exit(1)
		} else if err.Error() == "version requested by user" {
			ShowVersion()
			os.Exit(0)
		} else {
			panic(errors.Errorf("%v", err))
		}
	}

	// 即時終了する処理
	if len(args.ExportCode) > 0 {
		if p, err := filepath.Abs(args.ExportCode); err != nil {
			panic(fmt.Errorf("ソースコードの出力に失敗しました: %v\n", err))
			os.Exit(1)
		} else {
			args.ExportCode = filepath.ToSlash(p)
		}
		if err := exportSourceCode(args.ExportCode); err != nil {
			panic(fmt.Errorf("ソースコードの出力に失敗しました: %v\n", err))
			os.Exit(1)
		}
		os.Exit(0)
	}
}

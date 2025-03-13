package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// グローバル変数。
var (
	tempDir          string
	toolPaths        = make(map[string]string) // ツール名からパスへのマッピング
	initialized      = false
	binary_debug_log = false

	version  string = "debug build"   // makefileからビルドされると上書きされる。
	revision string = func() string { // {{{
		revision := ""
		modified := false
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					//return setting.Value
					revision = setting.Value
					if len(setting.Value) > 7 {
						revision = setting.Value[:7] // 最初の7文字にする
					}
				}
				if setting.Key == "vcs.modified" {
					modified = setting.Value == "true"
				}
			}
		}
		if modified {
			revision = "develop+" + revision
		}
		return revision
	}() // }}}
)

// init関数: ログの初期化。
func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func init() {
	SetupTool("busybox", "resources/busybox64u.exe")
	SetupTool("jq", "resources/jq.exe")
}

func example_tools() {
	// main関数の例
	// 起動時に各種ツールを展開
	if err := SetupTool("busybox", "resources/busybox64u.exe"); err != nil {
		log.Printf("エラー: Busyboxの初期化に失敗: %v\n", err)
		os.Exit(1)
	}

	// 別のツールも追加できます
	// err = SetupTool("other-tool", "resources/other-tool.exe")

	// ここで通常のアプリケーションロジックを実行
	// defer CleanupTools() は不要 - シグナルハンドラが処理します

	// busyboxコマンドの例
	if IsToolAvailable("busybox") {
		output, err := ExecuteTool("busybox", "ls", "-la")
		// または以下のように互換性関数も使用可能
		// output, err := ExecuteBusybox("ls", "-la")
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
		} else {
			fmt.Println("コマンド出力:")
			fmt.Println(output)
		}
	}

	// アプリケーションのメインロジックをここに記述
	// ... その他の処理

	// 明示的にクリーンアップする場合（通常は必要ありません）
	// CleanupTools()
}

func ShowHelp(post string) {
	buf := new(bytes.Buffer)
	parser.WriteHelp(buf)
	help := buf.String()
	help = strings.ReplaceAll(help, "display this help and exit", "ヘルプを出力する。")
	help = strings.ReplaceAll(help, "display version and exit", "バージョンを出力する。")
	fmt.Printf("%v\n", help)
	if len(post) != 0 {
		fmt.Println(post)
	}
	os.Exit(1)
}

func GetVersion() string {
	if len(revision) == 0 {
		// go installでビルドされた場合、gitの情報がなくなる。その場合v0.0.0.のように末尾に.がついてしまうのを避ける。
		return fmt.Sprintf("%v version %v", GetFileNameWithoutExt(os.Args[0]), version)
	} else {
		return fmt.Sprintf("%v version %v.%v", GetFileNameWithoutExt(os.Args[0]), version, revision)
	}
}

func ShowVersion() {
	fmt.Printf("%s\n", GetVersion())
	os.Exit(0)
}

func GetCurrentDir() string {
	ret, err := os.Getwd()
	if err != nil {
		panic(errors.Errorf("%v", err))
	}
	return filepath.ToSlash(ret)
}

func fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	if err != nil {
		return false
		//panic(errors.Errorf("%v", err))
	}
	return !os.IsNotExist(err)
}

func GetFileNameWithoutExt(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

// exportSourceCode は埋め込まれたソースコードを指定されたディレクトリに出力します
func exportSourceCode(outputDir string) error {
	// ディレクトリを作成
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %v", err)
	}

	// 埋め込みリソースディレクトリからソースファイルを取得
	srcFiles, err := AssetDir("resources/src")
	if err != nil {
		return fmt.Errorf("埋め込みソースコードの取得に失敗: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ソースコードを %s に出力します\n", outputDir)

	// 各ファイルを出力
	for _, filename := range srcFiles {
		data, err := Asset("resources/src/" + filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: %s の取得に失敗しました: %v\n", filename, err)
			continue
		}

		outPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "警告: %s の書き込みに失敗しました: %v\n", outPath, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s を出力しました\n", filename)
	}

	fmt.Fprintf(os.Stderr, "ソースコードの出力が完了しました\n")
	return nil
}

func SetDebugLogFlag(debug bool) {
	binary_debug_log = debug
}

// SetupTool は指定されたツールを一時ディレクトリに展開します
func SetupTool(toolName, assetPath string) error {

	// 一時ディレクトリの初期化
	if tempDir == "" {
		var err error
		tempDir, err = os.MkdirTemp("", "app_tools")
		if err != nil {
			return fmt.Errorf("一時ディレクトリの作成に失敗: %w", err)
		}

		// シグナルハンドラを設定してプログラム終了時にクリーンアップ
		setupCleanupHandler()
		initialized = true
	}

	// 埋め込まれたツールデータを取得
	toolData, err := Asset(assetPath)
	if err != nil {
		return fmt.Errorf("埋め込みファイル %s の取得に失敗: %w", assetPath, err)
	}

	// 一時ファイルとして書き出す
	toolPath := filepath.Join(tempDir, filepath.Base(assetPath))
	if err := os.WriteFile(toolPath, toolData, 0755); err != nil {
		return fmt.Errorf("%s の書き出しに失敗: %w", toolName, err)
	}

	// ツールパスを記録
	toolPaths[toolName] = toolPath

	if binary_debug_log {
		log.Printf("%s が展開されました: %s\n", toolName, toolPath)
	}
	return nil
}

// GetToolPath は指定されたツールのパスを取得します
func GetToolPath(toolName string) (string, bool) {
	path, exists := toolPaths[toolName]
	return path, exists
}

// IsToolAvailable は指定されたツールが利用可能かどうかを確認します
func IsToolAvailable(toolName string) bool {
	_, exists := toolPaths[toolName]
	return exists
}

// ExecuteTool は指定されたツールでコマンドを実行します
func ExecuteTool(toolName string, args ...string) (string, error) {
	toolPath, exists := toolPaths[toolName]
	if !exists {
		return "", fmt.Errorf("%s が初期化されていません。先にSetupToolを呼び出してください", toolName)
	}

	if binary_debug_log {
		log.Printf("exec.Command(%v, %#q)", toolPath, args)
	}

	cmd := exec.Command(toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s コマンド実行エラー: %w", toolName, err)
	}

	return string(output), nil
}

// CleanupTools は一時ディレクトリとツールを削除します
func CleanupTools() {
	if tempDir != "" {
		os.RemoveAll(tempDir)
		fmt.Println("一時ディレクトリを削除しました")
		tempDir = ""
		toolPaths = make(map[string]string)
		initialized = false
	}
}

// CleanupBusybox は一時ディレクトリを削除します（互換性のため）
func CleanupBusybox() {
	CleanupTools()
}

// setupCleanupHandler は終了シグナルを捕捉してクリーンアップを実行するハンドラを設定します
func setupCleanupHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n終了シグナルを受信しました")
		CleanupTools()
		os.Exit(0)
	}()
}

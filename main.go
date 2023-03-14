package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	xj "github.com/basgys/goxml2json"
)

func main() {

	flag.Parse()
	for i := 0; i < flag.NArg(); i++ {
		name := flag.Arg(i)
		file, err := os.Open(name) // *os.File
		defer file.Close()
		if err != nil {
			log.Println(err)
			continue
		}
		r := bufio.NewReader(file)
		jsonbb, err := xj.Convert(r)
		var outb bytes.Buffer
		json.Indent(&outb, jsonbb.Bytes(), "", "\t")
		output, err := os.Create(filepath.Join(filepath.Dir(name), filepath.Base(name)+".json"))
		defer output.Close()
		if err != nil {
			log.Println(err)
			continue
		}
		outb.WriteTo(output)
	}
}

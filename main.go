package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	xj "github.com/basgys/goxml2json"
)

func main() {

	flag.Parse()

	if flag.NArg() == 0 {
		b, err := readXml(bufio.NewReader(os.Stdin))
		if err != nil {
			log.Fatal(err)
		}
		if err := write(b, os.Stdout); err != nil {
			log.Fatal(err)
		}
	} else {
		for i := 0; i < flag.NArg(); i++ {
			name := flag.Arg(i)
			file, err := os.Open(name)
			defer file.Close()
			if err != nil {
				log.Println(err)
				continue
			}

			output, err := os.Create(filepath.Join(filepath.Dir(name), filepath.Base(name)+".json"))
			defer output.Close()
			if err != nil {
				log.Println(err)
				continue
			}
			b, err := readXml(bufio.NewReader(file))
			if err != nil {
				log.Println(err)
				continue
			}
			if err := write(b, output); err != nil {
				log.Println(err)
			}
		}
	}
}

func readXml(r io.Reader) (bytes.Buffer, error) {
	jsonbb, err := xj.Convert(r)
	if err != nil {
		return bytes.Buffer{}, err
	}
	var out bytes.Buffer
	json.Indent(&out, jsonbb.Bytes(), "", "\t")
	return out, nil
}

func write(data bytes.Buffer, w io.Writer) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if _, err := data.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

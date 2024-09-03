package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/terminal"
)

func main() {
	var (
		scanner = bufio.NewScanner(os.Stdin)
		input   strings.Builder
	)
	for scanner.Scan() {
		input.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading standard input:", err)
		os.Exit(1)
	}

	qrc, err := qrcode.New(input.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not generate QRCode", err)
		os.Exit(1)
	}

	if err = qrc.Save(terminal.New()); err != nil {
		fmt.Fprintln(os.Stderr, "save fail", err)
		os.Exit(1)
	}
}

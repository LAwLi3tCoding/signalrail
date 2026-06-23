package main

import (
	"context"
	"os"

	"github.com/LAwLi3tCoding/signalrail/internal/cli"
)

func main() { os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

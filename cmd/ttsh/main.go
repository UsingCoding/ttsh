package main

import (
	"context"
	"fmt"
	"os"

	appcli "github.com/usingcoding/ttsh/internal/cli"
)

func main() {
	cmd := appcli.New(appcli.Dependencies{})
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

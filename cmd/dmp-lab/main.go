// Command dmp-lab deploys and inspects a Debezium Platform lab environment.
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
)

// version is overwritten at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := fang.Execute(context.Background(), newRootCmd(), fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

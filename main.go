package main

import (
	"context"
	"log"
	"time"

	"hic/pkg/cli"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := cli.NewRootCommand(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}

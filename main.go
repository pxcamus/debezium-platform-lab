package main

import (
	"context"
	"log"

	"dbz-mage/ko/automation"
	mongodbsource "dbz-mage/ko/source/mongodb"
)

func main() {
	if err := automation.LoadEnv(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	mongodb, err := mongodbsource.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = mongodb.Close(context.Background())
	}()

	if err := mongodb.Wait(ctx); err != nil {
		log.Fatal(err)
	}

	if err := mongodb.Setup(ctx); err != nil {
		log.Fatal(err)
	}

	if err := mongodb.Populate(ctx); err != nil {
		log.Fatal(err)
	}
}

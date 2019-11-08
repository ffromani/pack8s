package main

import (
	"log"

	"github.com/fromanirh/pack8s/internal/pkg/version"

	"github.com/fromanirh/pack8s/cmd"
)

func main() {
	log.Printf("pack8s start - %s %s", version.VERSION, version.REVISION)
	defer log.Printf("pack8s done")
	cmd.Execute()
}

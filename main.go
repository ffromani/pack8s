package main

import (
	"log"

	"github.com/fromanirh/pack8s/cmd"
)

func main() {
	log.Printf("pack8s version DEV starting up")
	defer log.Printf("pack8s version DEV done")
	cmd.Execute()
}

package main

import (
	"log"
	"shakalizator/internal/pkg/app"
)

func main() {
	err := app.New()
	if err != nil {
		log.Fatal(err)
	}
}

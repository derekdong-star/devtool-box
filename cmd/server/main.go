package main

import (
	"log"

	"devtoolbox/internal/app"
)

func main() {
	if err := app.New().Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

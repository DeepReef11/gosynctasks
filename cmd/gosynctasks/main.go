package main

import (
	"fmt"
	"gosynctasks/internal/config"
	//"os"
	// "log"
	// "github.com/emersion/go-webdav/caldav"
)

func main() {
	fmt.Println("ok")

	// configObj := config.GetConfig()
	connector := config.GetConfig()
	fmt.Println(*connector)
	fmt.Println("End.")
}

package main

import (
	"fmt"
	"github.com/DeepReef11/gosynctasks/internal/config"
	//"os"
	// "log"
	// "github.com/emersion/go-webdav/caldav"
)

func main() {
	fmt.Println("ok")

	// configObj := config.GetConfig()
	connector := config.GetConnector()
	fmt.Println(*connector)
	fmt.Println("End.")
}

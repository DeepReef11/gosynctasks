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

	config := config.GetConfig()
	if config.Connector.Type == "nextcloud" {
		if nc, ok := config.Connector.(*connectors.NextcloudConnector); ok {
			fmt.Println(nc.Username) // 
		}
	}
	fmt.Println(*config)
	fmt.Println("End.")
}

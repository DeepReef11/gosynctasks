package main

import (
	"fmt"
	"gosynctasks/internal/config"
	// "gosynctasks/backend"
	//"os"
	"log"
	// "github.com/emersion/go-webdav/caldav"
)

func main() {
	connector := config.GetConfig()
	taskManager, err := connector.Connector.TaskManager()
	if err != nil {
		log.Fatalln("There was a problem with connector's task manager.")
	}
	tl, err := taskManager.GetTaskLists()
	if err != nil {
		fmt.Println("There was an error:")
		fmt.Println(err)
	}
	fmt.Println(tl)
	fmt.Println(tl[0].ID)
	tasks,err := taskManager.GetTasks(tl[0].ID)
	if err != nil {
		fmt.Println("There was an error while retrieving tasks:")
		fmt.Println(err)
	}
	fmt.Println(tasks)
	fmt.Println(*connector)
	fmt.Println("End.")
}

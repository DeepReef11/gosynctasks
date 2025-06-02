package main

import (
	"context"
	"fmt"
	"gosynctasks/internal/config"
	"os"
	// "gosynctasks/backend"
	"log"
	// "github.com/emersion/go-webdav/caldav"
	"database/sql"
	"github.com/urfave/cli/v3"
	_ "modernc.org/sqlite"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "tasks.db")
	if err != nil {
		return nil, err
	}

	// Create table
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS tasks (
            id TEXT PRIMARY KEY,
            content TEXT,
            status TEXT,
            created_at INTEGER,
            updated_at INTEGER
        )
    `)
	return db, err
}

func main() {

	(&cli.Command{}).Run(context.Background(), os.Args)
	config := config.GetConfig()
	taskManager, err := config.Connector.TaskManager()
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
	tasks, err := taskManager.GetTasks(tl[0].ID)
	if err != nil {
		fmt.Println("There was an error while retrieving tasks:")
		fmt.Println(err)
	}
	fmt.Println(tasks)

	tasks, err = taskManager.GetTasks(tl[0].ID)

	fmt.Println(tasks)
	fmt.Println(*config)
	fmt.Println("End.")
}

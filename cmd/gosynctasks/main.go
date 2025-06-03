package main

import (
	"context"
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"log"
	"os"
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
	config := config.GetConfig()
	filter := &backend.TaskFilter{}
	cmd := &cli.Command{
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			{
				Name:  "status",
				Usage: "filter task by status",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "done", Aliases: []string{"d"}},
					&cli.BoolFlag{Name: "todo", Aliases: []string{"t"}},
					&cli.BoolFlag{Name: "processing", Aliases: []string{"p"}},
					&cli.BoolFlag{Name: "cancelled", Aliases: []string{"c"}},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					var statuses []string

					if cmd.Bool("done") {
						statuses = append(statuses, "DONE")
					}
					if cmd.Bool("todo") {
						fmt.Println("TODO flag")
						statuses = append(statuses, "TODO")
					}
					if cmd.Bool("processing") {
						statuses = append(statuses, "PROCESSING")
					}
					if cmd.Bool("cancelled") {
						statuses = append(statuses, "CANCELLED")
					}

					if len(statuses) > 0 {
						filter.Statuses = &statuses
					}

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

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
	tasks, err := taskManager.GetTasks(tl[0].ID, filter)
	if err != nil {
		fmt.Println("There was an error while retrieving tasks:")
		fmt.Println(err)
	}
	fmt.Println(tasks)

	fmt.Println(*config)
	fmt.Println("End.")
}

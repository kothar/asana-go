package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"log"
	"net/http"
	"net/url"

	"bitbucket.org/mikehouston/asana-go"
	"bitbucket.org/mikehouston/asana-go/util"
)

var options struct {
	Token string `long:"token" description:"Personal Access Token used to authorize access to the API" env:"ASANA_TOKEN" required:"true"`

	Workspace []int64 `long:"workspace" short:"w" description:"Workspace to access"`
	Project   []int64 `long:"project" short:"p" description:"Project to access"`
	Task      []int64 `long:"task" short:"t" description:"Task to access"`

	Debug   bool   `short:"d" long:"debug" description:"Show debug information"`
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose output"`
}

func authenticate(req *http.Request) (*url.URL, error) {
	req.Header.Add("Authorization", "Bearer "+options.Token)
	return nil, nil
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if _, err := flags.Parse(&options); err != nil {
		return
	}

	// Create a client
	client := asana.NewClient(&http.Client{
		Transport: &http.Transport{
			Proxy: authenticate,
		},
	})
	if options.Debug {
		client.Debug = true
		client.DefaultOptions.Pretty = true
	}
	client.Verbose = options.Verbose

	// Load a task object
	if options.Task == nil {

		// Load a project object
		if options.Project == nil {

			// Load a workspace object
			if options.Workspace == nil {
				check(util.ListWorkspaces(client))
				return
			}

			for _, w := range options.Workspace {
				workspace := client.Workspace(w)
				check(util.ListProjects(workspace))
			}
			return
		}

		for _, p := range options.Project {
			project := client.Project(p)

			check(util.ListTasks(project))
		}
		return
	}

	for _, t := range options.Task {
		task := client.Task(t)
		check(task.Expand())

		fmt.Printf("Task %d: %s\n", task.ID, task.Name)
		fmt.Printf("  Completed: %v\n", task.Completed)
		if !task.Completed {
			fmt.Printf("  Due: %s\n", task.DueAt)
		}
		if task.Notes != "" {
			fmt.Printf("  Notes: %s\n", task.Notes)
		}

		// Get subtasks
		subtasks, nextPage, err := task.Subtasks()
		check(err)
		_ = nextPage

		for _, subtask := range subtasks {
			fmt.Printf("  Subtask %d: %s\n", subtask.ID, subtask.Name)
		}
	}
}

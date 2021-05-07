package main

import (
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"bitbucket.org/mikehouston/asana-go"
)

var options struct {
	Token string `long:"token" description:"Personal Access Token used to authorize access to the API" env:"ASANA_TOKEN" required:"true"`

	Workspace []string `long:"workspace" short:"w" description:"Workspace to access"`
	Project   []string `long:"project" short:"p" description:"Project to access"`
	Task      []string `long:"task" short:"t" description:"Task to access"`

	Attach     string `long:"attach" description:"Attach a file to a task"`
	AddSection string `long:"add-section" description:"Add a new section to a project"`

	Stories bool `long:"stories" description:"List stories for a task"`
	Clean   bool `long:"clean" description:"Clean all stories from a task"`

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
	client.DefaultOptions.Enable = []asana.Feature{asana.StringIDs, asana.NewSections, asana.NewTaskSubtypes}

	// Load a task object
	if options.Task == nil {

		// Load a project object
		if options.Project == nil {

			// Load a workspace object
			if options.Workspace == nil {
				check(ListWorkspaces(client))
				return
			}

			for _, w := range options.Workspace {
				workspace := &asana.Workspace{ID: w}
				check(ListProjects(client, workspace))
			}
			return
		}

		for _, p := range options.Project {
			project := &asana.Project{ID: p}

			if options.AddSection != "" {
				request := &asana.SectionBase{
					Name: options.AddSection,
				}

				_, err := project.CreateSection(client, request)
				check(err)
				return
			}

			fmtProject(client, project)
		}
		return
	}

	for _, t := range options.Task {
		task := &asana.Task{ID: t}
		check(task.Fetch(client))

		fmt.Printf("Task %s: %q\n", task.ID, task.Name)
		if options.Attach != "" {
			addAttachment(task, client)
			return
		}
		if options.Stories {
			listStories(task, client)
		}
		if options.Clean {
			cleanStories(task, client)
		}

		fmtTask(task, client)
	}
}

func listStories(task *asana.Task, client *asana.Client) {
	stories, _, _ := task.Stories(client)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for _, s := range stories {
		fmt.Printf("Story %s (%s):\n", s.ID, s.CreatedBy.Name)
		check(enc.Encode(s))
	}
}

func cleanStories(task *asana.Task, client *asana.Client) {
	stories, _, _ := task.Stories(client)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for _, s := range stories {
		check(s.Delete(client))
	}
}

func fmtProject(client *asana.Client, project *asana.Project) {
	fmt.Println("\nSections:")
	check(ListSections(client, project))
	fmt.Println("\nTasks:")
	check(ListTasks(client, project))
}

func fmtTask(task *asana.Task, client *asana.Client) {
	fmt.Printf("  Completed: %v\n", task.Completed)
	if task.Completed != nil && !*task.Completed {
		fmt.Printf("  Due: %s\n", task.DueAt)
	}
	if task.Notes != "" {
		fmt.Printf("  Notes: %q\n", task.Notes)
	}
	// Get subtasks
	subtasks, nextPage, err := task.Subtasks(client)
	check(err)
	_ = nextPage
	for _, subtask := range subtasks {
		fmt.Printf("  Subtask %s: %q\n", subtask.ID, subtask.Name)
	}
}

func addAttachment(task *asana.Task, client *asana.Client) {
	f, err := os.Open(options.Attach)
	check(err)
	defer f.Close()
	a, err := task.CreateAttachment(client, &asana.NewAttachment{
		Reader:      f,
		FileName:    f.Name(),
		ContentType: mime.TypeByExtension(filepath.Ext(f.Name())),
	})
	check(err)
	fmt.Printf("Attachment added: %+v", a)
}

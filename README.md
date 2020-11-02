# Asana API client for Go

This project implements an API client for the Asana REST API.

## Getting started

Here are some very brief examples of using the client.
There are comments in the code, but there
is a test application in [cmd/asana](cmd/asana) which
shows how some basic requests can be used.

To use a personal access token:
 
``` go
client := asana.NewClientWithAccessToken(token)
```

To use OAuth login, see the methods in [oauth.go](oauth.go).

To fetch workspace details:
``` go
w := &asana.Workspace{
  ID: "12345",
}

w.fetch(client)
```

To list tasks in a project:
``` go
p := &asana.Project{
  ID: "3456",
}

tasks, nextPage, err := p.Tasks(client, &asana.Options{Limit: 10})
```
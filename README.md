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

## Testing

`go test ./...` runs the unit tests, which mock the API. The integration tests in
[integration](integration) call the live API, creating and deleting their own objects
in a scratch workspace. They are skipped unless these environment variables are set:

| Variable | Purpose |
| --- | --- |
| `ASANA_TEST_PAT` | Personal access token for a test account |
| `ASANA_TEST_WORKSPACE` | gid of the scratch workspace the tests may write to |
| `ASANA_TEST_TEAM` | Optional: team to create projects in, if the workspace is an organization |

``` sh
ASANA_TEST_PAT=... ASANA_TEST_WORKSPACE=... go test -v -count=1 ./integration/
```

Tests of paid features, such as custom fields and dependencies, skip on a free workspace.

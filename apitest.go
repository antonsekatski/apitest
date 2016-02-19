package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Host string
	Port string
}

type Test struct {
	Before  *BA
	After   *BA
	Path    string
	Method  string
	Headers map[string]string
	Result  *Result
}

type BA struct {
	Cmd []string
}

type Result struct {
	Status int
	Body   interface{}
}

func (t *Test) MarshalResultBody() []byte {
	var i interface{}
	switch t.Result.Body.(type) {
	case string:
		// Normalize
		json.Unmarshal([]byte(t.Result.Body.(string)), &i)
	default:
		i = t.Result.Body
	}

	b, _ := json.Marshal(t.Result.Body)

	return b
}

func main() {
	app := cli.NewApp()
	app.Name = "apitest"
	app.Usage = "usage"
	app.Action = action

	app.Run(os.Args)
}

func action(c *cli.Context) {
	m := make(map[string]*Test)
	// Parse Yaml
	data, err := ioutil.ReadFile(c.Args().First())
	if err != nil {
		fmt.Println(err)
		return
	}
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		fmt.Println(err)
		return
	}

	normalize(m)

	runTests(m)
}

func normalize(m map[string]*Test) {
	for k, v := range m {
		spl := strings.Split(k, " ")
		v.Method = spl[0]
		v.Path = spl[1]
	}
}

func runTests(m map[string]*Test) {
	for _, v := range m {
		req, _ := http.NewRequest(v.Method, "http://127.0.0.1:7071"+v.Path, nil)

		color.Cyan("-------------------------------------")
		color.Cyan("Run test: %s %s", v.Method, v.Path)

		if v.Before != nil {
			runBA(v.Before)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			color.Red("Error doing request: %s", err)
		}

		// Check status
		if v.Result.Status != 0 && res.StatusCode != v.Result.Status {
			color.Yellow("Status doesn't match: should %d, get: %d", v.Result.Status, res.StatusCode)
		}

		// Check body
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			color.Red("Error reading body: %s", err)
		}
		res.Body.Close()

		// Normalize
		var t1 interface{}
		json.Unmarshal(body, &t1)
		resBody, _ := json.Marshal(t1)

		testBody := v.MarshalResultBody()

		if !bytes.Equal(resBody, testBody) {
			color.Red("Body doesn't match:")
			color.Green("- Should: %+v", v.Result.Body)
			color.Yellow("- Get: %+v", t1)
		} else {
			color.Green("OK")
		}

		if v.After != nil {
			runBA(v.After)
		}
	}
}

func runBA(ba *BA) {
	if len(ba.Cmd) == 0 {
		return
	}

	for _, v := range ba.Cmd {
		// Split command to parts and head
		parts := strings.Fields(v)
		head := parts[0]
		parts = parts[1:len(parts)]
		// Run command
		if err := exec.Command(head, parts...).Run(); err != nil {
			color.Red("Can't run before/after command: %s", err)
		}
	}
}

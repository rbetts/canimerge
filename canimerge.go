package main

import (
    "fmt"
    "encoding/json"
    "flag"
    "net/http"
    "io/ioutil"
    "log"
)

var detail bool
var branch string
func init() {
    flag.BoolVar(&detail, "detail", false, "Print detailed job status.")
    flag.StringVar(&branch, "branch", "", "Branch name you're considering merging.")
}

type Job struct {
    Name string
    Url string
    Color string
}

type View struct {
    // Description string
    Jobs []Job
    Url string
}

func main() {
    flag.Parse()
    if len(branch) == 0 {
        log.Fatal("Please specify a branch name.")
    }

    view := decodeView(getViewJSON("A-master"))
    if isViewBlue(view) {
        fmt.Printf("Master is blue.\n")
    } else {
        fmt.Printf("Master is NOT blue.\n")
    }

    view = decodeView(getViewJSON("branch-" + branch))
    if isViewBlue(view) {
        fmt.Printf("Branch is blue.\n")
    } else {
        fmt.Printf("Branch is NOT blue.\n")
    }

}

func getViewJSON(viewname string) []byte {
    url := "http://ci/view/" + viewname + "/api/json?pretty=true"
    res, err := http.Get(url)
    if err != nil {
        log.Fatal("Error retrieving data about " + viewname + ". ", err)
    }
    body,err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Fatal("Error reading HTTP response retrieving " + viewname + ". ", err)
    }
    return body
}

func decodeView(body []byte) View {
    var view View
    err := json.Unmarshal(body, &view)
    if (err != nil) {
        log.Fatal("Error unmarshalling json. ", err)
    }
    return view
}

func isViewBlue(view View) bool {
    for _, job := range view.Jobs {
        if detail {
            fmt.Printf("Job %s is %s\n", job.Name, job.Color)
        }
        if job.Color != "blue" {
            return false
        }
    }
    return true
}





package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var detail bool
var branch string
var debug bool
var checkout bool

func init() {
	flag.BoolVar(&detail, "detail", false, "Print detailed job status.")
	flag.BoolVar(&debug, "debug", false, "Print retreived json (verbose)")
	flag.BoolVar(&checkout, "checkout", false, "Use the currently checked-out git branch instead of <branch>")
}

/*
 * Top level branch view structures
 */

// Job is one test suite or configuration contained within a view
type Job struct {
	Name  string
	Url   string
	Color string
}

// View is the top-level branch view containing a set of Jobs.
// A-master, for example, is a View
type View struct {
	/* Description string */
	Jobs []Job
	Url  string
}

/*
 * Detailed test result structures
 */

// TestReportCase is one junit test in a suite
type TestReportCase struct {
	ClassName string
	Duration  float64
	Name      string
	Status    string
}

// TestReportSuite is testReport/childReports/result/suites
type TestReportSuite struct {
	Cases []TestReportCase
	Name  string
}

// TestChildReportsResult is testReport/childReports/result {}
type TestChildReportsResult struct {
	Duration  float64
	FailCount int
	PassCount int
	SkipCount int
	Suites    []TestReportSuite
}

// TestChildReports testReport/childReports {}
type TestChildReports struct {
	Result TestChildReportsResult
}

// TestReport is root of http://ci/.../testReport
type TestReport struct {
	FailCount  int
	SkipCount  int
	TotalCount int
	// some jobs contain sub-view childreports
	ChildReports []TestChildReports
	// others directly contain suites
	Suites []TestReportSuite
}

/*
 * The munging...
 */

func main() {
	flag.Parse()
	if checkout {
		branch = resolveCurrentGitBranch()
	} else {
		branch = flag.Arg(0)
	}
	if len(branch) == 0 {
		log.Print("\nUsage: canimerge [--debug] [--detail] <-checkout | branchname>\n")
		flag.PrintDefaults()
		return
	}
	checkBranch("A-master", "master")
	if detail {
		fmt.Printf("\n")
	}
	checkBranch("branch-"+branch, branch)
	if detail {
		fmt.Printf("\n")
	}
}

func resolveCurrentGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--symbolic-full-name",
		"--abbrev-ref", "HEAD").Output()
	if err != nil {
		log.Fatal("Can not resolve --checkout branch name. ", err)
	}
	return strings.TrimSpace(string(out))
}

func checkBranch(branch, display string) {
	url := "http://ci/view/" + branch + "/api/json?pretty=true"
	view := decodeView(getJSON(url))
	if isViewBlue(view, branch) {
		fmt.Printf(">> PASS: " + display + ".\n")
	} else {
		fmt.Printf(">> FAIL: " + display + ".\n")
	}
}

// getJSON retrieves json from url via HTTP GET
func getJSON(url string) []byte {
	res, err := http.Get(url)
	defer res.Body.Close()
	if err != nil {
		log.Fatal("Error retrieving data from "+url+". ", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error reading HTTP response from "+url+". ", err)
	}
	if debug {
		fmt.Printf("URL: %s BODY: %s", url, body)
	}
	return body
}

func decodeView(body []byte) View {
	var view View
	if err := json.Unmarshal(body, &view); err != nil {
		log.Fatal("Error unmarshalling json. ", err)
	}
	return view
}

func isViewBlue(view View, branch string) (retval bool) {
	retval = true
	for _, job := range view.Jobs {
		if job.Color == "blue" {
			if detail {
				fmt.Printf("PASS: %s\n", job.Name)
			}
		} else if job.Color == "blue_anime" {
			if detail {
				fmt.Printf("PASS (in progress): %s\n", job.Name)
			}
		} else if strings.Contains(job.Color, "aborted") {
			if detail {
				fmt.Printf("ABORTED: %s\n", job.Name)
			}
		} else if job.Color == "red_anime" {
			if detail {
				fmt.Printf("FAIL (in progress): %s\n", job.Name)
				printBranchFailureDetails(branch, job.Name)
			}
			retval = false
		} else {
			if detail {
				fmt.Printf("FAIL: %s\n", job.Name)
				printBranchFailureDetails(branch, job.Name)
			}
			retval = false
		}
	}
	return
}

func printBranchFailureDetails(branch string, job string) {
	url := "http://ci/view/" + branch + "/job/" +
		job + "/lastCompletedBuild/testReport/api/json?pretty=true"

	body := getJSON(url)
	var testReport TestReport
	if err := json.Unmarshal(body, &testReport); err != nil {
		fmt.Printf("\tNo detail results available for " + job + ".\n")
		return
	}

	var printed bool = false

	// Print results for sub-views.
	for _, childReports := range testReport.ChildReports {
		for _, suite := range childReports.Result.Suites {
			for _, test := range suite.Cases {
				if test.Status == "FAILED" {
					printed = true
					fmt.Printf("\t%s %s failed\n", suite.Name, test.Name)
				}
			}
		}
	}

	// Print results for directly contained suites
	for _, suite := range testReport.Suites {
		for _, test := range suite.Cases {
			if test.Status == "FAILED" {
				printed = true
				fmt.Printf("\t%s %s failed\n", suite.Name, test.Name)
			}
		}
	}

	if debug && !printed {
		fmt.Printf("DEBUG BODY:\n%s", body)
	}
}

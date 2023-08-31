package main

import (
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/xanzy/go-gitlab"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Project struct {
	name      string
	projectId string
}

var projects = []Project{
	{name: "Test", projectId: "123456"},
}

func loadReleases(privateToken string, projectId string, numberOfProjects float64) []*gitlab.Release {
	git, err := gitlab.NewClient(privateToken)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	opt := &gitlab.ListReleasesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: int(math.Min(numberOfProjects, 10)),
			Page:    1,
		},
	}
	var releases []*gitlab.Release
	var currentItems int = 0
	for {
		rs, resp, er := git.Releases.ListReleases(projectId, opt, nil)
		if er != nil {
			log.Fatalf("Failed to create client: %v", er)
		}
		releases = append(releases, rs...)
		releasesLength := len(releases)
		currentItems += opt.PerPage
		// Exit the loop when we've seen all pages.
		if resp.NextPage == 0 || releasesLength >= int(numberOfProjects) {
			break
		}

		// Update the page number to get the next page.
		if currentItems+opt.PerPage > int(numberOfProjects) {
			opt.PerPage = currentItems + opt.PerPage - int(numberOfProjects)
		}
		opt.Page = resp.NextPage
	}

	return releases
}

func removeDuplicates(array []string) []string {
	m := make(map[string]string)
	for _, x := range array {
		m[x] = x
	}
	var ClearedArr []string
	for x, _ := range m {
		ClearedArr = append(ClearedArr, x)
	}
	return ClearedArr
}

func printTable(projectName string, releases []*gitlab.Release) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Tag", "Name", "CreatedAt", "QS_Tag", "Content", "Link"})
	for _, release := range releases {
		var ticketNumber = ""
		description := release.Description
		if strings.Contains(description, "renovate/all") {
			ticketNumber = "Renovate"
		} else {
			r := regexp.MustCompile("FN[-_\\d]?\\w+|HF[-_\\d]?\\w+")
			ticketNumber = strings.Join(
				removeDuplicates(r.FindAllString(description, -1)), ", ")
		}
		t.AppendRow(table.Row{
			release.TagName,
			release.Name,
			release.CreatedAt.Format("02.01.06 - 15:04"),
			fmt.Sprintf("QS_%s", release.Commit.ShortID),
			ticketNumber,
			release.Links.Self,
		})
	}
	log.Print(projectName)
	t.Render()
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [numberOfReleases=3]\n To run this script, you need to expose a GitlabPrivateToken as Env\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	gitlabToken := os.Getenv("GITLAB_AUTH_TOKEN")

	if gitlabToken == "" {
		flag.Usage()
	}
	args := flag.Args()
	var numberOfProjects float64 = 3
	if len(args) > 0 {
		var err error
		numberOfProjects, err = strconv.ParseFloat(args[0], 64)
		if err != nil {
			panic("Could not read NumberOfReleases")
		}
	}

	var wg sync.WaitGroup

	for _, project := range projects {
		wg.Add(1)
		go func(project Project) {
			defer wg.Done()
			releases := loadReleases(gitlabToken, project.projectId, numberOfProjects)
			printTable(project.name, releases)
		}(project)
	}
	wg.Wait()
}

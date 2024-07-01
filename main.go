package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Manifest struct {
	XMLName  xml.Name  `xml:"manifest"`
	Remotes  []Remote  `xml:"remote"`
	Projects []Project `xml:"project"`
}

type Remote struct {
	Name     string `xml:"name,attr"`
	Fetch    string `xml:"fetch,attr"`
	Revision string `xml:"revision,attr"`
}

type Project struct {
	Path   string `xml:"path,attr"`
	Name   string `xml:"name,attr"`
	Remote string `xml:"remote,attr"`
	Groups string `xml:"groups,attr,omitempty"`
}

func main() {
	var mergeAll bool
	var pushFlag bool
	var branchOrTag string

	flag.BoolVar(&mergeAll, "merge-all", false, "Merge all repositories defined in the manifest")
	flag.BoolVar(&pushFlag, "p", false, "Push repositories after merge")
	flag.StringVar(&branchOrTag, "b", "master", "Branch or tag to merge from")
	flag.Parse()

	manifestFile := "manifest.xml"
	manifest, err := readManifest(manifestFile)
	if err != nil {
		fmt.Printf("Error reading manifest file: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		os.Exit(1)
	}

	if mergeAll {
		for _, project := range manifest.Projects {
			if project.Groups == "aosp-platform" {
				err := mergeRepository(project, branchOrTag, cwd, manifest)
				if err != nil {
					fmt.Printf("Error merging %s: %v\n", project.Name, err)
				}
			}
		}
	}

	if pushFlag {
		for _, project := range manifest.Projects {
			err := pushRepository(project, cwd, manifest)
			if err != nil {
				fmt.Printf("Error pushing %s: %v\n", project.Name, err)
			}
		}
	}
}

func readManifest(filename string) (Manifest, error) {
	var manifest Manifest
	file, err := os.Open(filename)
	if err != nil {
		return manifest, err
	}
	defer file.Close()

	err = xml.NewDecoder(file).Decode(&manifest)
	if err != nil {
		return manifest, err
	}

	return manifest, nil
}

func mergeRepository(project Project, branchOrTag string, cwd string, manifest Manifest) error {
	repoPath := filepath.Join(cwd, project.Path)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found in path: %s", repoPath)
	}

	cmd := exec.Command("git", "-C", repoPath, "checkout", branchOrTag)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "-C", repoPath, "pull", "https://android.googlesource.com"+project.Path, branchOrTag)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func pushRepository(project Project, cwd string, manifest Manifest) error {
	repoPath := filepath.Join(cwd, project.Path)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found in path: %s", repoPath)
	}

	remoteURL := getRemoteURL(project, project.Revision, manifest)

	cmd := exec.Command("git", "-C", repoPath, "push", remoteURL)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func getRemoteURL(project Project, branchOrTag string, manifest Manifest) string {
	var fetchURL string

	for _, remote := range manifest.Remotes {
		if remote.Name == project.Remote {
			fetchURL = remote.Fetch
			break
		}
	}

	if fetchURL == "" {
		return ""
	}

	return fetchURL + project.Path + " " + branchOrTag
}

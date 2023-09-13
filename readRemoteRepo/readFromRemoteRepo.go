package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/pkg/git"
)

var (
	//github url
	gitHubRepo = "github"
	github     = "https://github.com/openshift-kni/cnf-features-deploy/ztp/source-crs?ref=release-4.13"

	//gitlab url
	gitLabRepo    = "gitlab"
	gitlab_master = "https://gitlab.cee.redhat.com/sahasan/source-cr-project/source-crs?ref=main"
	gitlab_branch = "https://gitlab.cee.redhat.com/sahasan/source-cr-project/source-crs?ref=version_4.14"

	//local url running in nginx container
	//	localRepo       = "http://localhost:8888/content/source-crs"
	localRepo       = "http://10.88.0.17/content/source-crs"
	remoteLocalPath = "./sourceCR"

	// github: storageNS; gitlab: SriovSubscription
	//	filename  = []string{"MachineConfigAcceleratedStartup.yaml", "test2.yaml", "test.yaml", "StorageNS.yaml", "sabbir.yaml", "SriovSubscription_gitlab.yaml", "ClusterLogCatSource.yaml"}
	filename  = []string{"MachineConfigAcceleratedStartup.yaml", "StorageNS.yaml", "SriovSubscription_gitlab.yaml"}
	url_paths = []string{github, localRepo, remoteLocalPath, gitlab_branch}
	rawURL    = "https://raw.githubusercontent.com"
)

var absolutePath, filePath string
var clonedDir, cachedUrl []string
var repo *git.RepoSpec
var err error
var notClone = "/notCloned"

const httpScheme = "http://"
const httpsScheme = "https://"
const prefix = "./"

func splitYamls(yamls []byte) ([][]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(yamls))
	var resources [][]byte

	for {
		var resIntf interface{}
		err := decoder.Decode(&resIntf)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Check that resIntf is not nil in order to mitigate appending an empty
		// object as a result of redundant trailing seperator(s) "---""
		if resIntf != nil {
			resBytes, err := yaml.Marshal(resIntf)

			if err != nil {
				return nil, err
			}

			resources = append(resources, resBytes)
		}
	}
	return resources, nil
}

func transformURL(url, fileName string) (string, error) {

	var newURL string
	testString, err := git.NewRepoSpecFromUrl(url)
	if err != nil {
		fmt.Println(err)
		return newURL, err
	}
	//	fmt.Printf("hostname:  %+v\n", testString)
	hostname := reflect.ValueOf(testString).Elem().FieldByName("host")
	repoName := reflect.ValueOf(testString).Elem().FieldByName("orgRepo")
	branch := reflect.ValueOf(testString).Elem().FieldByName("ref")
	repoPath := reflect.ValueOf(testString).Elem().FieldByName("path")

	//	test := hostname.String()

	switch {
	case strings.Contains(hostname.String(), gitHubRepo):
		newURL = rawURL + "/" + repoName.String() + "/" + branch.String() + "/" + repoPath.String() + "/" + fileName
		/*
			fmt.Printf("newURL:  %s\n", newURL)
			fmt.Printf("hostname:  %s\n", hostname)
			fmt.Printf("repopath %s\n", repoName)
			fmt.Printf("branch %s\n", branch)
			fmt.Printf("ref %s\n", branch)
		*/
	case strings.Contains(hostname.String(), gitLabRepo):
		newURL = hostname.String() + repoName.String() + "/-/raw/" + branch.String() + "/" + repoPath.String() + "/" + fileName

	default:
		newURL = url + "/" + fileName

	}

	return newURL, nil
}

func readContentRemote(link string) ([]byte, error) {

	var content []byte
	resp, err := http.Get(link)
	if err != nil {
		fmt.Printf("http error ** %s **\n", err)
		return content, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return content, fmt.Errorf("URL : '%s' not found\n", link)
	}

	defer resp.Body.Close()

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Reading error from URL: ** %s **\n", err)
		return content, err
	}

	return content, nil
}

type Status struct {
	Url      string
	fileName []string
}

func updateDatabase(p string, file string, st []Status) []Status {

	if len(st) == 0 {
		for _, v := range url_paths {
			st = append(st, Status{
				Url:      v,
				fileName: []string{},
			})
		}
	}

	for i, v := range st {
		if v.Url == p {
			v.fileName = append(v.fileName, file)
			st[i].fileName = v.fileName
		}
	}

	return st
}

type NFound struct {
	found    bool
	fileName []string
}

func updateFileStatus(file string, t bool, fStatus []NFound) []NFound {
	//	fmt.Printf("%%--%% Received file:%s and found: %v, current struct value %v %%--%%\n", file, t, fStatus)
	if len(fStatus) == 0 {
		fStatus = append(fStatus, NFound{
			found:    true,
			fileName: []string{},
		}, NFound{
			found:    false,
			fileName: []string{},
		})
	}

	// test =  NFound{found: true, fileName: []string{},found: true, fileName: []string{}}
	for i, v := range fStatus {
		if v.found == t {
			v.fileName = append(v.fileName, file)
			fStatus[i].fileName = v.fileName
		}
	}
	//	fmt.Printf("%%--%% value of updated struct : %v %%--%%\n", fStatus)
	return fStatus

}

func WriteFile(content, path string) {

	fileWrite, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = fmt.Fprintln(fileWrite, content)
	if err != nil {
		fmt.Println(err)
		fileWrite.Close()
		return
	}
	err = fileWrite.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func Readfile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func CacheContent(url, filePath string) (string, *git.RepoSpec, error) {

	var sourceCRLocation string

	repo, err := git.NewRepoSpecFromUrl(url)
	if err != nil {
		fmt.Println(err)
		return sourceCRLocation, repo, err
	}

	// read from the file to verify if the url and path
	//  is already cloned

	relativePath := reflect.ValueOf(repo).Elem().FieldByName("path")
	cloneDirInfo := reflect.ValueOf(repo).Elem().FieldByName("cloneDir")

	cloned := isCloned(filePath, url, relativePath.String())

	if !cloned {
		err = git.ClonerUsingGitExec(repo)
		if err != nil {
			fmt.Println(err)
			return sourceCRLocation, repo, err
		}
	}
	err = git.ClonerUsingGitExec(repo)
	if err != nil {
		fmt.Println(err)
		return sourceCRLocation, repo, err
	}

	sourceCRLocation = repo.AbsPath()
	// write in the /tmp dir
	if cloneDirInfo.String() != notClone {
		writeString := url + "," + sourceCRLocation
		fmt.Printf("Writing in file %v\n", filePath)
		WriteFile(writeString, filePath)
	}

	fmt.Printf("** absolutepath after: %v\n", sourceCRLocation)
	fmt.Printf("** Repo info: %+v\n", repo)

	return sourceCRLocation, repo, nil
}

func main() {

	// loop through filename, and inside loop through the destination path/link
	writeFileOutput := "output.txt"
	writeClonedInfo := "cloned.txt"

	startTime := time.Now()

	var updated []Status
	var fStatus []NFound

	for _, f := range filename {

	Loop:
		for i, p := range url_paths {
			//	if strings.Contains(p, httpScheme) || strings.Contains(p, httpsScheme) {

			switch {
			case strings.Contains(p, httpScheme) || strings.Contains(p, httpsScheme):

				/* when we read live from remote URL
				   we need to compute the final url
				*/
				// // compute final URL
				// finalUrl, err := transformURL(p, f)
				// if err != nil {
				// 	fmt.Println(err)
				// }

				/* when we git clone remote URL
				   we need to cache the content
				*/

				if !strings.Contains(p, gitHubRepo) && !strings.Contains(p, gitLabRepo) {
					filePath = p + "/" + f
					color.Red("Reading from local Container")
				} else {
					if !contains(cachedUrl, p) {
						absolutePath, repo, err = CacheContent(p, writeClonedInfo)
						if err != nil {
							fmt.Println(err)
						}
						color.Green("Cloned Repo at %v", clonedDir)
						cachedUrl = append(cachedUrl, p)
						clonedDir = append(clonedDir, repo.CloneDir().String())
						color.Green("Cached URL %v", cachedUrl)
					}
					filePath = absolutePath + "/" + f
				}

				/* when we read live from remote URL
				   we need to call http.get method (Curl) to read the content
				*/
				// // read Content
				// fileByte, err := readContentRemote(finalUrl)

				/* when we git clone remote URL
				   we need to read the content from local file
				*/

				fileByte, err := Readfile(filePath)

				if err == nil {
					fmt.Printf("file %s found in path %s\n", f, filePath)
					//	updated = updateDatabase(p, f, updated)
					updated = updateDatabase(p, f, updated)
					fStatus = updateFileStatus(f, true, fStatus)
					_ = fileByte
					//	fmt.Println(string(fileByte))
					color.Cyan("%s read from link %s\n", f, p)
					WriteFile(fmt.Sprintf("- %s read from link %s", f, filePath), writeFileOutput)
					break Loop

				} else if err != nil && i == (len(url_paths)-1) {
					fStatus = updateFileStatus(f, false, fStatus)
					fmt.Printf("%s was not found in any path/URL\n", f)
					WriteFile(fmt.Sprintf("- %s was not found in any URL", f), writeFileOutput)
					color.Red("%s was tried to read from link %s\n", f, p)
				}

			case strings.HasPrefix(p, prefix):
				//	fmt.Printf("path: %s is not a remote url\n", p)
				currentDir, err := os.Getwd()
				if err != nil {
					fmt.Printf("err is: %s\n", err)
				}
				path := currentDir + "/" + p + "/" + f

				fileByte, err := os.ReadFile(path)
				if err == nil {
					updated = updateDatabase(p, f, updated)
					fStatus = updateFileStatus(f, true, fStatus)
					_ = fileByte
					//	fmt.Println(string(fileByte))
					color.Cyan("%s read from path %s\n", f, p)
					WriteFile(fmt.Sprintf("- %s read from link %s", f, p), writeFileOutput)
					break Loop
				}
				if errors.Is(err, os.ErrNotExist) && i == (len(url_paths)-1) {
					fStatus = updateFileStatus(f, false, fStatus)
					fmt.Printf("%s was not found in path\n", p)
					WriteFile(fmt.Sprintf("- %s was not found in path", p), writeFileOutput)
					color.Red("%s was tried to read from link %s\n", f, p)

				} else {
					fmt.Printf("last error is %v\n", err)
					color.Red("%s was tried to read from link %s\n", f, p)
				}

			}

		}
	}

	removeDir(clonedDir)
	elapsedTime := time.Since(startTime)
	//	ReadFIlebyLine("output.txt")
	fmt.Printf("cloned dir list %v\n", clonedDir)
	fmt.Printf("Time execution: %v\n", elapsedTime)
	printMessage(fStatus, updated)

	//	github     = "https://github.com/openshift-kni/cnf-features-deploy/ztp/source-crs?ref=release-4.13"

}

func isCloned(fileName, url, rlPath string) bool {

	fmt.Printf("isCloned received fileName :%s\n", fileName)

	check_file, err := os.Stat(fileName)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if check_file.Size() == 0 {
		fmt.Printf("file size is %d\n", check_file.Size())
		return false
	}

	var fileString []byte
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND, 0644)

	if err != nil {
		fmt.Println(err)
		return false
	}
	file.Close()

	fileString, err = os.ReadFile(fileName)

	testString := strings.Split(string(fileString), ",")

	for _, v := range testString {
		if v == url || strings.Contains(v, rlPath) {
			return true
		}
	}

	color.Red("The value from file is: %v\n", testString)
	return false

}

func removeDir(clonedDir []string) {
	for _, v := range clonedDir {
		err := os.RemoveAll(v)
		if err != nil {
			fmt.Printf("Couldn't clean the repo, err: %v\n", err)
		}
	}
}

func printMessage(fStatus []NFound, updated []Status) {
	fmt.Println("\n")

	fmt.Printf("** File status struct** %v\n", fStatus)

	fmt.Println("\n")

	fmt.Println(strings.Repeat("*", 27))
	color.Cyan("Reading all files are done")
	fmt.Println(strings.Repeat("*", 27))

	// fileWrite, err := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	fmt.Println(strings.Repeat("-", 143))
	w := tabwriter.NewWriter(os.Stdout, 10, 0, 0, ' ', tabwriter.Debug)
	//w := tabwriter.NewWriter(fileWrite, 10, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Order\t Url Link\t File Name\t")
	for i, v := range updated {
		fmt.Fprintln(w, i, "\t", v.Url, "\t", v.fileName, "\t")
	}

	w.Flush()
	fmt.Println(strings.Repeat("-", 143))

	fmt.Println("\n")
	fmt.Println(strings.Repeat("-", 140))
	w = tabwriter.NewWriter(os.Stdout, 10, 0, 0, ' ', tabwriter.Debug)
	//	w = tabwriter.NewWriter(fileWrite, 10, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "File Status\t File Name\t")
	for _, v := range fStatus {
		fmt.Fprintln(w, v.found, "\t", v.fileName, "\t")
	}

	w.Flush()
	fmt.Println(strings.Repeat("-", 140))
	fmt.Println("\n")

}

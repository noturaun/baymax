package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	mvnSedArgs = []Args{
		{flag: "-e", value: "1,9d"},
		{flag: "-e", value: "s/\\[INFO\\]//g"},
		{flag: "-e", value: "s/---//g"},
		{flag: "-e", value: "s/[|+\\]//g"},
		{flag: "-e", value: "/BUILD SUCCESS/d"},
		{flag: "-e", value: "/Total time/d"},
		{flag: "-e", value: "/Finished/d"},
		{flag: "-e", value: "s/ //g"},
		{flag: "-e", value: "s/:compile//g"},
		{flag: "-e", value: "s/:test//g"},
		{flag: "-e", value: "s/:provided//g"},
		{flag: "-e", value: "/^$/d"},
		{flag: "-e", value: "s/^-//g"},
	}
)

const (
	MAVEN        = "maven"
	POM          = "pom.xml"
	NPM          = "npm"
	PACKAGE_JSON = "package.json"
)

type Args struct {
	flag  string
	value string
}

func Check(target string) (*bytes.Buffer, string) {
	path = target
	checkPath()
	checkDependencyFile()
	return spawn()
}

type Dependency struct {
	GroupID    string       `json:"groupId"`
	ArtifactID string       `json:"artifactId"`
	Version    string       `json:"version"`
	Scope      string       `json:"scope,omitempty"`
	Children   []Dependency `json:"children,omitempty"`
}

func pipeline(stdout *bytes.Buffer, cmds ...*exec.Cmd) (err error) {
	var stderr bytes.Buffer
	pipes := make([]*io.PipeWriter, len(cmds)-1)
	i := 0
	for ; i < len(cmds)-1; i++ {
		in, out := io.Pipe()
		cmds[i].Stdout = out
		cmds[i].Stderr = &stderr
		cmds[i+1].Stdin = in
		pipes[i] = out
	}
	cmds[i].Stdout = stdout
	cmds[i].Stderr = &stderr

	return call(cmds, pipes)
}

func call(cmds []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if cmds[0].Process == nil {
		if err = cmds[0].Start(); err != nil {
			return err
		}
	}
	if len(cmds) > 1 {
		if err = cmds[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = call(cmds[1:], pipes[1:])
			}
		}()
	}
	return cmds[0].Wait()
}

func spawn() (*bytes.Buffer, string) {
	var cmd *exec.Cmd
	if format == MAVEN {
		cmd = exec.Command("mvn", "dependency:tree")
	} else if format == NPM {
		cmd = exec.Command("npm", "ls", "--all")
	} else {
		fmt.Println("Unsupported operation")
		os.Exit(1)
	}

	cmd.Dir = path

	var cmds []*exec.Cmd

	cmds = append(cmds, cmd)
	for _, args := range mvnSedArgs {
		cmds = append(cmds, exec.Command("sed", args.flag, args.value))
	}

	var buff bytes.Buffer
	if err := pipeline(&buff,
		cmds...,
	); err != nil {
		fmt.Println("Unable to run command pipeline")
		os.Exit(1)
	}

	//if _, err := io.Copy(os.Stdout, &buff); err != nil {
	//	fmt.Println("Unable to copy data to std out")
	//	os.Exit(1)
	//}

	return &buff, format

}

func checkPath() {
	if path != "" {
		if !strings.Contains(path, POM) && !strings.Contains(path, PACKAGE_JSON) {
			format = checkDependencyFile()
		} else {
			if strings.Contains(path, PACKAGE_JSON) {
				path = strings.Replace(path, PACKAGE_JSON, "", 1)
				format = NPM
			} else if strings.Contains(path, POM) {
				path = strings.Replace(path, POM, "", 1)
				format = MAVEN
			} else {
				fmt.Println("Unsupported")
				os.Exit(1)
			}
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Cannot read directory, %s\n", err)
			os.Exit(1)
		}
		path = cwd
		format = checkDependencyFile()
	}
}

func checkDependencyFile() string {
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Cannot read directory, %s\n", err)
		os.Exit(1)
	}
	for _, e := range entries {
		if e.Name() == POM {
			return MAVEN
		}
		if e.Name() == PACKAGE_JSON {
			return NPM
		}
	}
	return ""
}

func checkMvn() {
	cmd := exec.Command("mvn", "dependency:tree")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error running mvn command:", err)
		return
	}

	dependencies := parseDependencies(out.String())
	jsonString, err := json.MarshalIndent(dependencies, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	fmt.Println(string(jsonString))
}

type NpmDependency struct {
	Version      string                    `json:"version"`
	Resolved     string                    `json:"resolved,omitempty"`
	Dependencies map[string]*NpmDependency `json:"dependencies,omitempty"`
}

func checkNpm() {
	cmd := exec.Command("npm", "ls", "--silent", "--json")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running npm command:", err)
		return
	}

	var dependencies map[string]interface{}
	if err := json.Unmarshal(out, &dependencies); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	jsonString, err := json.MarshalIndent(dependencies["dependencies"], "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	fmt.Println(string(jsonString))
}

func parseDependencies(output string) []Dependency {
	lines := strings.Split(output, "\n")
	var rootDependencies []Dependency
	var stack []*Dependency

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		depth := strings.Count(line, "|") + strings.Count(line, "\\") + strings.Count(line, "+")
		trimmed := strings.TrimSpace(line)
		dep := parseDependencyLine(trimmed)

		if depth == 0 {
			rootDependencies = append(rootDependencies, dep)
			stack = []*Dependency{&rootDependencies[len(rootDependencies)-1]}
		} else {
			for len(stack) > depth {
				stack = stack[:len(stack)-1]
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, dep)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}

	return rootDependencies
}

func parseDependencyLine(line string) Dependency {
	parts := strings.Fields(line)
	artifact := parts[0]
	details := strings.Split(artifact, ":")
	groupID := details[0]
	artifactID := details[1]
	version := details[2]
	scope := ""
	if len(details) > 3 {
		scope = details[3]
	}
	return Dependency{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		Scope:      scope,
	}
}

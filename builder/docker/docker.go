package docker

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/appc/spec/schema/types"
	"github.com/sgotti/baci/builder/util"

	"github.com/docker/docker/builder/parser"
)

const (
	defaultPathEnv = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

type DockerBuilder struct {
	root      string
	sourceDir string
	workDir   string
	node      *parser.Node
	cmd       []string
	user      string
	group     string
	env       map[string]string
	ports     []types.Port
}

func NewDockerBuilder(root string, sourceDir string) (*DockerBuilder, error) {
	b := &DockerBuilder{
		root:      root,
		sourceDir: sourceDir,
		env:       make(map[string]string),
	}
	err := b.parseDockerFile()
	if err != nil {
		return nil, fmt.Errorf("error parsing Dockerfile: %v", err)
	}
	return b, nil
}

func (b *DockerBuilder) GetBaseImage() (string, error) {
	for _, n := range b.node.Children {
		if n.Value == "from" {
			n := n.Next
			if n == nil {
				return "", fmt.Errorf("missing parameter")
			}
			return n.Value, nil
		}
	}
	return "", nil
}

func (b *DockerBuilder) GetExec() []string {
	return b.cmd
}

func (b *DockerBuilder) GetUser() string {
	return b.user
}

func (b *DockerBuilder) GetGroup() string {
	return b.group
}

func (b *DockerBuilder) GetWorkDir() string {
	return b.workDir
}

func (b *DockerBuilder) GetPorts() ([]types.Port, error) {
	count := 0
	for _, n := range b.node.Children {
		if n.Value == "expose" {
			n := n.Next
			if n == nil {
				return nil, fmt.Errorf("missing parameter")
			}
			v := n.Value
			port, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, err
			}
			portName, err := types.NewACName(fmt.Sprintf("port%d", count))
			if err != nil {
				return nil, err
			}
			b.ports = append(b.ports, types.Port{Name: *portName, Protocol: "tcp", Port: uint(port)})
		}
	}
	return b.ports, nil
}

func (b *DockerBuilder) GetEnv() map[string]string {
	return b.env
}

func makeEnvString(env map[string]string) []string {
	envString := []string{}
	for k, v := range env {
		envString = append(envString, fmt.Sprintf("%s=%s", k, v))
	}
	return envString
}

var (
	// `\\\\+|[^\\]|\b|\A` - match any number of "\\" (ie, properly-escaped backslashes), or a single non-backslash character, or a word boundary, or beginning-of-line
	// `\$` - match literal $
	// `[[:alnum:]_]+` - match things like `$SOME_VAR`
	// `{[[:alnum:]_]+}` - match things like `${SOME_VAR}`
	tokenEnvInterpolation = regexp.MustCompile(`(\\|\\\\+|[^\\]|\b|\A)\$([[:alnum:]_]+|{[[:alnum:]_]+})`)
	// this intentionally punts on more exotic interpolations like ${SOME_VAR%suffix} and lets the shell handle those directly
)

// handle environment replacement.
// This is the same function used by docker
func (b *DockerBuilder) replaceEnv(str string) string {
	for _, match := range tokenEnvInterpolation.FindAllString(str, -1) {
		idx := strings.Index(match, "\\$")
		if idx != -1 {
			if idx+2 >= len(match) {
				str = strings.Replace(str, match, "\\$", -1)
				continue
			}

			prefix := match[:idx]
			stripped := match[idx+2:]
			str = strings.Replace(str, match, prefix+"$"+stripped, -1)
			continue
		}

		match = match[strings.Index(match, "$"):]
		matchKey := strings.Trim(match, "${}")

		for k, v := range b.env {
			if k == matchKey {
				str = strings.Replace(str, match, v, -1)
				break
			}
		}
	}
	return str
}

func (b *DockerBuilder) Cmd(node *parser.Node) error {
	params := []string{}
	for p := node.Next; p != nil; p = p.Next {
		params = append(params, p.Value)
	}

	args := []string{}
	if len(params) == 0 {
		return fmt.Errorf("missing parameters")
	}
	if node.Attributes != nil && node.Attributes["json"] {
		// Exec form
		args = params
	} else {
		args = append([]string{"/bin/sh", "-c"}, params[0])
	}

	b.cmd = args
	return nil
}
func (b *DockerBuilder) Run(node *parser.Node) error {
	params := []string{}
	for p := node.Next; p != nil; p = p.Next {
		params = append(params, p.Value)
	}

	args := []string{}
	if len(params) == 0 {
		return fmt.Errorf("missing params")
	}
	if node.Attributes != nil && node.Attributes["json"] {
		// Exec form
		args = params
	} else {
		args = append([]string{"/bin/sh", "-c"}, params[0])
	}

	env := append([]string{"PATH=" + defaultPathEnv}, makeEnvString(b.env)...)
	cmd := exec.Cmd{
		Env:    env,
		Dir:    b.workDir,
		Path:   args[0],
		Args:   args,
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	log.Printf("running command: %s\n", strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running command: %v", err)
	}
	return nil
}

func (b *DockerBuilder) SetWorkDir(node *parser.Node) error {
	n := node.Next
	if n == nil {
		return fmt.Errorf("missing parameter")
	}
	w := b.replaceEnv(n.Value)
	if filepath.IsAbs(w) {
		b.workDir = filepath.Join(b.root, w)
	} else {
		b.workDir = filepath.Join(b.workDir, w)
	}

	return nil
}

func (b *DockerBuilder) Add(node *parser.Node) error {
	// TODO(sgotti) handle other cases like:
	// download urls
	n := node.Next
	if n == nil {
		return fmt.Errorf("missing parameter")
	}
	source := b.replaceEnv(n.Value)

	n = n.Next
	if n == nil {
		return fmt.Errorf("missing parameter")
	}
	v := b.replaceEnv(n.Value)

	var dest string
	if filepath.IsAbs(n.Value) {
		dest = filepath.Join(b.root, v)
	} else {
		dest = filepath.Join(b.workDir, v)
	}

	isTar := false
	ext := filepath.Ext(source)
	if ext == ".tar" {
		isTar = true
	}
	if util.StringInSlice(ext, []string{".gz", ".bz2", ".xz"}) {
		ext := filepath.Ext(strings.TrimRight(source, ext))
		if ext == ".tar" {
			isTar = true
		}
	}

	if isTar {
		log.Printf("Extracting %s in %s\n", source, dest)
		err := util.ExtractACI(filepath.Join(b.sourceDir, source), dest)
		if err != nil {
			return fmt.Errorf("error extracting source file %s: %v", source, err)
		}
		return nil
	}

	files, err := filepath.Glob(filepath.Join(b.sourceDir, source))
	if err != nil {
		return fmt.Errorf("error adding source file %s: %v", source, err)
	}
	for _, f := range files {
		_, err := util.CopyFile(f, dest)
		if err != nil {
			return fmt.Errorf("error adding source file %s: %v", source, err)
		}
		log.Printf("Adding %s to %s\n", source, dest)
	}
	return nil
}

func (b *DockerBuilder) Env(node *parser.Node) error {
	env := make(map[string]string)
	var key, value string
	isKey := true
	for p := node.Next; p != nil; p = p.Next {
		if isKey {
			key = p.Value
			isKey = false
		} else {
			value = b.replaceEnv(p.Value)
			env[key] = value
			isKey = true
		}
	}

	for k, v := range env {
		b.env[k] = v
	}

	return nil
}

func (b *DockerBuilder) User(node *parser.Node) error {
	n := node.Next
	if n == nil {
		return fmt.Errorf("missing parameter")
	}
	b.user = b.replaceEnv(n.Value)
	return nil
}

func (b *DockerBuilder) parseDockerFile() error {
	df, err := os.Open(filepath.Join(b.sourceDir, "Dockerfile"))
	if err != nil {
		return fmt.Errorf("cannot open Dockerfile: %v", err)
	}
	defer df.Close()
	b.node, err = parser.Parse(df)
	if err != nil {
		return fmt.Errorf("error parsing Dockerfile: %v", err)
	}
	return nil
}

func (b *DockerBuilder) Unsupported(n *parser.Node) error {
	command := n.Value
	log.Printf("Command %s not supported, ignoring (this can lead to wrong behavior of the next commands)\n", command)
	return nil
}

func (b *DockerBuilder) Build() error {
	cmdMap := map[string]func(*parser.Node) error{
		"from":       nil,
		"maintainer": nil,
		"expose":     nil,
		"workdir":    b.SetWorkDir,
		"add":        b.Add,
		"copy":       b.Add, // TODO(sgotti) now COPY is handled like ADD.
		"cmd":        b.Cmd,
		"run":        b.Run,
		"env":        b.Env,
		"user":       b.User,
	}

	// TODO(sgotti) handle:
	// ENTRYPOINT
	// VOLUME
	// ONBUILD (how???)
	for _, n := range b.node.Children {
		command := n.Value
		f, ok := cmdMap[command]
		if !ok {
			b.Unsupported(n)
		}
		if f != nil {
			err := f(n)
			if err != nil {
				return fmt.Errorf("error execting Dockerfile command %q: %v", n.Original, err)
			}
		}
	}
	return nil
}

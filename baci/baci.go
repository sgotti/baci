// Copyright 2015 Simone Gotti <simone.gotti@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/discovery"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/sgotti/baci/builder/docker"
	"github.com/sgotti/baci/common"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/coreos/rocket/cas"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/jessevdk/go-flags"
)

var (
	defaultBaciImage string // either set by linker, or guessed in init()
)

func init() {
	// if not set by linker, try discover the directory baci is running
	// from, and assume the default baci.aci is stored alongside it.
	if defaultBaciImage == "" {
		if exePath, err := os.Readlink("/proc/self/exe"); err == nil {
			defaultBaciImage = filepath.Join(filepath.Dir(exePath), "baci.aci")
		}
	}
}

var opts struct {
	StoreDir      string `long:"storedir" description:"Cas store dir to use" default:"/var/lib/baci"`
	RktPath       string `long:"rktpath" description:"Rocket executable path" required:"true"`
	OutFilePath   string `short:"o" long:"outfile" description:"The filename of the generated ACI" required:"true"`
	AppNameLabels string `short:"n" long:"name" description:"The ACI's name and labels in the format \"example.com/reduce-worker:1.0.2,os=linux,arch=amd64,anotherlabel=value\"" required:"true"`
	Args          struct {
		SourceDir string `positional-arg-name:"sourcedir" description:"The directory containing the build scripts (for example the Dockerfile)"`
	} `positional-args:"true" required:"true"`
}

func die(s string, i ...interface{}) {
	s = fmt.Sprintf(s, i...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(s, "\n"))
	os.Exit(1)
}

func downloadImage(aciURL string, ds *cas.Store) (string, error) {
	rem, ok, err := ds.GetRemote(aciURL)
	if err != nil {
		return "", err
	}
	if !ok {
		rem = cas.NewRemote(aciURL, "")
		_, aciFile, err := rem.Download(*ds, nil)
		if err != nil {
			return "", err
		}
		defer os.Remove(aciFile.Name())

		rem, err = rem.Store(*ds, aciFile)
		if err != nil {
			return "", err
		}
	}
	return rem.BlobKey, nil
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	rktPath := opts.RktPath
	sourceDir, err := filepath.Abs(opts.Args.SourceDir)
	if err != nil {
		die("error: %v", err)
	}

	App, err := discovery.NewAppFromString(opts.AppNameLabels)
	if err != nil {
		die("error: %v", err)
	}

	outFilePath := opts.OutFilePath
	outDir, err := filepath.Abs(filepath.Dir(outFilePath))
	if err != nil {
		die("error: %v", err)
	}
	outFile := filepath.Base(outFilePath)

	// TODO(sgotti) as this program exists after running rocket, no one is removing this dir
	dataDir, err := ioutil.TempDir("", "bacidata")
	if err != nil {
		die("error: %v", err)
	}

	b, err := docker.NewDockerBuilder("/", sourceDir)
	if err != nil {
		die("error: %v", err)
	}

	baseImage, err := b.GetBaseImage()
	if err != nil {
		die("error: %v", err)
	}

	log.Printf("baseImage: %s\n", baseImage)

	if baseImage != "" && baseImage != "scratch" {
		baseACIPath := filepath.Join(dataDir, "base.aci")

		ds, err := cas.NewStore(opts.StoreDir)
		if err != nil {
			die("error: %v", err)
		}

		url := fmt.Sprintf("docker://%s", baseImage)

		key, err := downloadImage(url, ds)
		if err != nil {
			die("error: %v", err)
		}

		log.Printf("image downloaded")
		r, err := ds.ReadStream(key)
		if err != nil {
			die("error: %v", err)
		}
		defer r.Close()

		mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		fh, err := os.OpenFile(baseACIPath, mode, 0644)
		if err != nil {
			die("error: %v", err)
		}
		defer fh.Close()
		_, err = io.Copy(fh, r)
		if err != nil {
			die("error: %v", err)
		}
		log.Printf("image written to %s", baseACIPath)
	}

	// Write config data for the baci builder
	labels := types.Labels{}
	for k, v := range App.Labels {
		name, err := types.NewACName(k)
		if err != nil {
			die("wrong label name: %v", err)
		}
		labels = append(labels, types.Label{Name: *name, Value: v})
	}
	configData := &common.ConfigData{
		OutFile: outFile,
		AppName: App.Name,
		Labels:  labels,
	}
	configDataJson, err := json.Marshal(configData)
	err = ioutil.WriteFile(filepath.Join(dataDir, "configdata"), []byte(configDataJson), 0644)
	if err != nil {
		die("error: %v", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	sourceVol := fmt.Sprintf("-volume=source,kind=host,source=%s", sourceDir)
	dataVol := fmt.Sprintf("-volume=data,kind=host,source=%s", dataDir)
	destVol := fmt.Sprintf("-volume=dest,kind=host,source=%s", outDir)
	volumesArgs := []string{sourceVol, destVol, dataVol}
	cmdArgs := append([]string{rktPath}, "run")
	cmdArgs = append(cmdArgs, volumesArgs...)
	cmdArgs = append(cmdArgs, defaultBaciImage)
	cmd := exec.Cmd{
		Env:    os.Environ(),
		Path:   cmdArgs[0],
		Args:   cmdArgs,
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting the Builder container: %v", err)
	}

	builderChan := make(chan error)
	go func() {
		builderChan <- cmd.Wait()
	}()

	for {
		select {
		case err := <-builderChan:
			if err != nil {
				log.Fatalf("Builder container exited with error: %v", err)
			}
			log.Printf("Builder container finished.")
			os.Exit(0)

		case sig := <-signalChan:
			log.Printf("Signal %v received", sig)
			cmd.Process.Signal(syscall.SIGKILL)
		}
	}

}

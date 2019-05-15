// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/process"

	"github.com/mcuadros/go-version"
)

// Version return this package's current version
func Version() string {
	return "0.4.2"
}

var (
	// Debug enables verbose logging on everything.
	// This should be false in case Gogs starts in SSH mode.
	Debug = false
	// Prefix the log prefix
	Prefix = "[git-module] "
	// GitVersionRequired is the minimum Git version required
	GitVersionRequired = "1.7.2"

	// GitExecutable is the command name of git
	// Could be updated to an absolute path while initialization
	GitExecutable = "git"

	gitVersion string
)

func log(format string, args ...interface{}) {
	if !Debug {
		return
	}

	fmt.Print(Prefix)
	if len(args) == 0 {
		fmt.Println(format)
	} else {
		fmt.Printf(format+"\n", args...)
	}
}

// BinVersion returns current Git version from shell.
func BinVersion() (string, error) {
	if len(gitVersion) > 0 {
		return gitVersion, nil
	}

	stdout, err := NewCommand("version").Run()
	if err != nil {
		return "", err
	}

	fields := strings.Fields(stdout)
	if len(fields) < 3 {
		return "", fmt.Errorf("not enough output: %s", stdout)
	}

	// Handle special case on Windows.
	i := strings.Index(fields[2], "windows")
	if i >= 1 {
		gitVersion = fields[2][:i-1]
		return gitVersion, nil
	}

	gitVersion = fields[2]
	return gitVersion, nil
}

func init() {
	absPath, err := exec.LookPath(GitExecutable)
	if err != nil {
		panic(fmt.Sprintf("Git not found: %v", err))
	}
	GitExecutable = absPath

	gitVersion, err := BinVersion()
	if err != nil {
		panic(fmt.Sprintf("Git version missing: %v", err))
	}
	if version.Compare(gitVersion, GitVersionRequired, "<") {
		panic(fmt.Sprintf("Git version not supported. Requires version > %v", GitVersionRequired))
	}

	// Git requires setting user.name and user.email in order to commit changes.
	for configKey, defaultValue := range map[string]string{"user.name": "Gitea", "user.email": "gitea@fake.local"} {
		if stdout, stderr, err := process.GetManager().Exec("git.Init(get setting)", GitExecutable, "config", "--get", configKey); err != nil || strings.TrimSpace(stdout) == "" {
			// ExitError indicates this config is not set
			if _, ok := err.(*exec.ExitError); ok || strings.TrimSpace(stdout) == "" {
				if _, stderr, gerr := process.GetManager().Exec("git.Init(set "+configKey+")", "git", "config", "--global", configKey, defaultValue); gerr != nil {
					panic(fmt.Sprintf("Failed to set git %s(%s): %s", configKey, gerr, stderr))
				}
			} else {
				panic(fmt.Sprintf("Failed to get git %s(%s): %s", configKey, err, stderr))
			}
		}
	}

	// Set git some configurations.
	if _, stderr, err := process.GetManager().Exec("git.Init(git config --global core.quotepath false)",
		GitExecutable, "config", "--global", "core.quotepath", "false"); err != nil {
		panic(fmt.Sprintf("Failed to execute 'git config --global core.quotepath false': %s", stderr))
	}
}

// Fsck verifies the connectivity and validity of the objects in the database
func Fsck(repoPath string, timeout time.Duration, args ...string) error {
	// Make sure timeout makes sense.
	if timeout <= 0 {
		timeout = -1
	}
	_, err := NewCommand("fsck").AddArguments(args...).RunInDirTimeout(timeout, repoPath)
	return err
}

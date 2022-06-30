package main

import (
	"bytes"
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/pelletier/go-toml"
	"os"
	"os/exec"
	"strings"
)

func getGitTagsForHead() ([]string, error) {
	stdout, err := runCmd("git", "tag", "--points-at", "HEAD")
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(stdout)), nil
}

func getGitTagVersion() (string, error) {
	_, err := runCmd("git", "fetch", "--tags")
	if err != nil {
		return "", fmt.Errorf("could not fetch tags: %v", err)
	}

	stdout, err := runCmd("git", "tag", "-l")
	if err != nil {
		return "", err
	}
	newestGitTagVersion, err := semver.Make("0.0.0")
	if err != nil {
		return "", err
	}

	for _, tag := range strings.Fields(string(stdout)) {
		gitTagVersion, err := semver.Make(tag)
		if err != nil {
			// ignore parsing error on tag, look for a new tag that is a semver
			continue
		}

		if gitTagVersion.GT(newestGitTagVersion) {
			newestGitTagVersion = gitTagVersion
		}
	}

	return newestGitTagVersion.String(), nil
}

func getSetupCfgVersion() (string, error) {
	config, err := toml.LoadFile("pyproject.toml")
	if err != nil {
		return "", err
	}
	project := config.Get("project")
	return project.(*toml.Tree).Get("version").(string), nil
}

func configureGit() error {
	stdout, err := runCmd("git", "config", "user.name")
	if err != nil {
		return err
	}
	if len(stdout) == 0 {
		_, err := runCmd("git", "config", "--global", "user.name", "github-actions[bot]")
		if err != nil {
			return err
		}
	}
	stdout, err = runCmd("git", "config", "user.email")
	if err != nil {
		return err
	}
	if len(stdout) == 0 {
		_, err := runCmd("git", "config", "--global", "user.email", "github-actions[bot]@users.noreply.github.com")
		if err != nil {
			return err
		}
	}
	return nil
}

func runCmd(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	stdout := outb.String()
	stderr := errb.String()
	if err != nil {
		fmt.Printf("failed to run command '%s %s', got stdout:\n%s\nstderr:\n%s\n", name, strings.Join(arg, " "), stdout, stderr)
		return "", fmt.Errorf("failed to run command: %v", err)
	}
	return stdout, nil
}

func main() {
	_, err := runCmd("git", "config", "--global", "--add", "safe.directory", "/github/workspace")
	if err != nil {
		panic(err)
	}

	setupCfgVersion, err := getSetupCfgVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println(setupCfgVersion)
	gitTagVersion, err := getGitTagVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println(gitTagVersion)
	gitTagsForHead, err := getGitTagsForHead()
	if err != nil {
		panic(err)
	}
	for _, tag := range gitTagsForHead {
		if setupCfgVersion == tag {
			fmt.Printf("already tagged %s on HEAD\n", setupCfgVersion)
			os.Exit(0)
		}
	}
	newVersion := ""
	if semver.MustParse(setupCfgVersion).GT(semver.MustParse(gitTagVersion)) {
		newVersion = setupCfgVersion
	} else {
		ver := semver.MustParse(setupCfgVersion)
		ver.Patch = ver.Patch + 1
		newVersion = ver.String()
	}
	fmt.Printf("new version to be published is %s", newVersion)
	err = updateTagAndSetupCfg(newVersion)
	if err != nil {
		panic(err)
	}
}

func updateTagAndSetupCfg(newVersion string) error {
	err := configureGit()
	if err != nil {
		return err
	}

	config, err := toml.LoadFile("pyproject.toml")
	if err != nil {
		return err
	}
	project := config.Get("project")
	version := project.(*toml.Tree).Get("version").(string)

	if version != newVersion {
		project.(*toml.Tree).Set("version", newVersion)
		err := writeToml("pyproject.toml", config)
		if err != nil {
			return err
		}
		_, err = runCmd("git", "add", "pyproject.toml")
		if err != nil {
			return err
		}
		_, err = runCmd("git", "commit", "-m", fmt.Sprintf("update version to %s in pyproject.toml", newVersion))
		if err != nil {
			return err
		}
	}
	fmt.Printf("git tag %s\n", newVersion)
	_, err = runCmd("git", "tag", newVersion)
	if err != nil {
		return err
	}
	githubRef := os.Getenv("GITHUB_REF")
	branch := strings.TrimPrefix(githubRef, "refs/heads/")
	fmt.Printf("git push --tags origin %s\n", branch)
	_, err = runCmd("git", "push", "--tags", "origin", branch)
	if err != nil {
		return err
	}
	return nil
}

func writeToml(path string, config *toml.Tree) error {
	t, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, t, 0644)
}

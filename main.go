package main

import (
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/pelletier/go-toml"
	"os"
	"os/exec"
	"strings"
)

func getGitTagsForHead() ([]string, error) {
	cmd := exec.Command("git", "tag", "--points-at", "HEAD")
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(stdout)), nil
}

func getGitTagVersion() (string, error) {
	cmd := exec.Command("git", "fetch", "--tags")
	_, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not fetch tags: %v", err)
	}

	cmd = exec.Command("git", "tag", "-l")
	stdout, err := cmd.Output()
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
	cmd := exec.Command("git", "config", "user.name")
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	if len(stdout) == 0 {
		cmd := exec.Command("git", "config", "--global", "user.name", "github-actions[bot]")
		_, err := cmd.Output()
		if err != nil {
			return err
		}
	}
	cmd = exec.Command("git", "config", "user.email")
	stdout, err = cmd.Output()
	if err != nil {
		return err
	}
	if len(stdout) == 0 {
		cmd := exec.Command("git", "config", "--global", "user.email", "github-actions[bot]@users.noreply.github.com")
		_, err := cmd.Output()
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
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
	updateTagAndSetupCfg(newVersion)
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
		cmd := exec.Command("git", "add", "pyproject.toml")
		_, err = cmd.Output()
		if err != nil {
			return err
		}
		cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("update version to %s in pyproject.toml", newVersion))
		_, err = cmd.Output()
		if err != nil {
			return err
		}
	}
	cmd := exec.Command("git", "tag", newVersion)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	githubRef := os.Getenv("GITHUB_REF")
	branch := strings.TrimPrefix(githubRef, "refs/heads/")
	cmd = exec.Command("git", "push", "--tags", "origin", branch)
	_, err = cmd.Output()
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

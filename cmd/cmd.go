package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
)

// Version - klarista cli version
var Version = "latest"

func getAutoFlags(override bool) string {
	if override || os.Getenv("CI") != "" {
		return "-auto-approve -compact-warnings"
	}
	return ""
}

func getVarFileFlags(inputIds []string) string {
	return strings.Join(
		cast.ToStringSlice(
			funk.Map(inputIds, func(id string) string {
				return fmt.Sprintf(`-var-file "inputs/%s"`, id)
			}),
		),
		" ",
	)
}

func shell(command string, args ...string) {
	filteredArgs := funk.Compact(args).([]string)
	cmd := exec.Command(command, filteredArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if <-done {
			return
		}

		sig := <-sigs
		Logger.Debug("GOT SIGNAL ", sig)

		if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
			if err := cmd.Process.Kill(); err != nil {
				Logger.Fatal("Failed to kill process: ", err)
			}
		}
	}()

	Logger.Debugf("%s %s", command, strings.Join(filteredArgs, " "))

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	done <- true
}

func useWorkDir(wd string, cb func()) {
	// Get the pwd
	originalWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	wd, err = filepath.Abs(wd)
	if err != nil {
		panic(err)
	}

	if wd == originalWd {
		Logger.Debugf(`Already in WD %s`, wd)
		cb()
	} else {
		// Change to the target wd
		Logger.Debugf(`Using WD %s`, wd)
		if err = os.Chdir(wd); err != nil {
			panic(err)
		}

		defer func() {
			// Return to the original wd
			Logger.Debugf(`Returning to WD %s`, originalWd)
			if err = os.Chdir(originalWd); err != nil {
				panic(err)
			}
		}()

		// Do work
		cb()
	}
}

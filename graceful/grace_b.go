// +build !windows
//
// Copyright 2016 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graceful

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func graceSignal() {
	// subscribe to SIGINT signals
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	defer func() {
		os.Exit(0)
	}()
	sig := <-ch
	signal.Stop(ch)
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM:
		Shutdown()
	case syscall.SIGUSR2:
		Reboot()
	}
}

// Reboot all the frame process gracefully.
// Notes: Windows system are not supported!
func Reboot(timeout ...time.Duration) {
	log.Infof("rebooting process...")

	var (
		ppid     = os.Getppid()
		graceful = true
	)
	contextExec(timeout, "reboot", func(ctxTimeout context.Context) <-chan struct{} {
		endCh := make(chan struct{})
		go func() {
			defer close(endCh)

			var reboot = true

			if preCloseFunc != nil {
				if err := preCloseFunc(); err != nil {
					log.Errorf("[reboot-preClose] %s", err.Error())
					graceful = false
				}
			}

			// Starts a new process passing it the active listeners. It
			// doesn't fork, but starts a new process using the same environment and
			// arguments as when it was originally started. This allows for a newly
			// deployed binary to be started.
			_, err := startProcess()
			if err != nil {
				log.Errorf("[reboot-startNewProcess] %s", err.Error())
				reboot = false
			}

			// shut down
			graceful = shutdown(ctxTimeout, "reboot") && graceful
			if !reboot {
				if graceful {
					log.Errorf("process reboot failed, but shut down gracefully!")
				} else {
					log.Errorf("process reboot failed, and did not shut down gracefully!")
				}
				os.Exit(-1)
			}
		}()

		return endCh
	})

	// Close the parent if we inherited and it wasn't init that started us.
	if ppid != 1 {
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			log.Errorf("[reboot-killOldProcess] %s", err.Error())
			graceful = false
		}
	}

	if graceful {
		log.Infof("process are rebooted gracefully.")
	} else {
		log.Infof("process are rebooted, but not gracefully.")
	}
}

var allProcFiles = []*os.File{os.Stdin, os.Stdout, os.Stderr}

// SetExtractProcFiles sets extract proc files for only reboot.
// Notes: Windows system are not supported!
func SetExtractProcFiles(extractProcFiles []*os.File) {
	for _, f := range extractProcFiles {
		var had bool
		for _, ff := range allProcFiles {
			if ff == f {
				had = true
				break
			}
		}
		if !had {
			allProcFiles = append(allProcFiles, f)
		}
	}
}

// In order to keep the working directory the same as when we started we record
// it at startup.
var originalWD, _ = os.Getwd()

// startProcess starts a new process passing it the active listeners. It
// doesn't fork, but starts a new process using the same environment and
// arguments as when it was originally started. This allows for a newly
// deployed binary to be started. It returns the pid of the newly started
// process when successful.
func startProcess() (int, error) {
	for _, f := range allProcFiles {
		defer f.Close()
	}

	// Use the original binary location. This works with symlinks such that if
	// the file it points to has been changed we will use the updated symlink.
	argv0, err := exec.LookPath(os.Args[0])
	if err != nil {
		return 0, err
	}

	process, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   originalWD,
		Env:   os.Environ(),
		Files: allProcFiles,
	})
	if err != nil {
		return 0, err
	}
	return process.Pid, nil
}

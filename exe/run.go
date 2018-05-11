package exe

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

type logger interface {
	Info(...interface{})
	Debug(...interface{})
	Error(...interface{})
}

// RunOutput executes the command in a bash process capturing the output and
// returning it in the string slice
func RunOutput(log logger, wd, cmd string, args ...string) ([]string, int) {
	var output []string

	out := make(chan string)
	rangeDone := make(chan bool)
	go func() {
		for t := range out {
			output = append(output, t)
		}
		rangeDone <- true
	}()

	status := Run(log, out, nil, wd, cmd, args...)

	<-rangeDone

	return output, status
}

// Run executes the command in a bash process
func Run(log logger, out chan string, env []string, wd, cmd string, args ...string) int {

	log.Info("Exec Cmd: ", cmd, " Args: ", args)

	if wd != "" {
		// make sure working directory is in place
		if err := os.MkdirAll(wd, 0700); err != nil {
			log.Error(err)
			return 1
		}
	}

	eCmd := exec.Command(cmd, args...)

	eCmd.Env = os.Environ()
	eCmd.Env = append(eCmd.Env, env...)

	// this is mandatory
	eCmd.Dir = wd
	log.Info("In working directory: ", eCmd.Dir)

	out <- cmd + " " + strings.Join(args, " ")
	out <- ""

	sOut, err := eCmd.StdoutPipe()
	if err != nil {
		log.Error("getting stdout", err)
		return 1
	}

	eOut, err := eCmd.StderrPipe()
	if err != nil {
		log.Error("getting stderr", err)
		return 1
	}

	// safely aggregate both to a single reader
	pr, pw := io.Pipe()

	// copy both to out and wait for them to close
	// start these before starting the cmd, to be sure we capture all output
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		if c, e := io.Copy(pw, eOut); e != nil {
			log.Error(e, c)
		}
		wg.Done()
	}()

	go func() {
		if c, e := io.Copy(pw, sOut); e != nil {
			log.Error(e, c)
		}
		wg.Done()
	}()

	// start scanning from the common pipe
	scanDone := make(chan bool)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			out <- scanner.Text()
		}
		if e := scanner.Err(); e != nil {
			out <- "scanning output failed with: " + e.Error()
		}
		scanDone <- true
	}()

	log.Debug("Exec starting")
	err = eCmd.Start()
	if err != nil {
		log.Error("start failed", err)
		out <- err.Error()
		out <- ""
		close(out)
		return 1
	}

	go func() {
		wg.Wait()
		pw.Close()
	}()

	log.Debug("Exec waiting")
	err = eCmd.Wait()

	// wait for scanner to fully complete
	<-scanDone
	close(out)

	log.Debug("exec cmd complete")

	if err != nil {
		log.Error("Command failed ", err)
		exitCode := 1
		if msg, ok := err.(*exec.ExitError); ok {
			if status, ok := msg.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
				log.Info("exit status: ", exitCode)
			}
		}
		// we prefer to return 0 for good or one for bad
		return exitCode
	}

	log.Info("Executing command succeeded")
	return 0
}

package exe

import (
	"bufio"
	"io"
	"os/exec"
	"syscall"
)

type logger interface {
	Info(...interface{})
	Debug(...interface{})
	Error(...interface{})
	Infof(format string, args ...interface{})
}

// Run executes the command in a bash process capturing the output and returning it in the string slice
func RunOutput(log logger, cmd, args, wd string) ([]string, int) {
	pr, pw := io.Pipe()
	var output []string

	scanDone := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			t := scanner.Text()
			output = append(output, t)
		}
		if err := scanner.Err(); err != nil {
			output = append(output, "scanning output failed with: "+err.Error())
		}
		scanDone <- true
	}()

	status := Run(log, cmd, args, wd, pw)
	<-scanDone // make sure the scanner has exited
	return output, status
}

// Run executes the command in a bash process
func Run(log logger, cmd, args, wd string, out io.WriteCloser) int {

	log.Infof("Exec Cmd: <%s> Args: <%s> ", cmd, args)
	argStr := cmd + " " + args
	eCmd := exec.Command("bash", "-c", argStr)

	// this is mandatory
	eCmd.Dir = wd
	log.Info("In working directory: ", eCmd.Dir)

	// out can be nil - it is only set for the first executing thread
	if out != nil {
		out.Write([]byte(wd + "$ " + argStr + "\n\n"))

		sout, err := eCmd.StdoutPipe()
		if err != nil {
			log.Info(err)
			return 1
		}
		eout, err := eCmd.StderrPipe()
		if err != nil {
			log.Error(err)
			return 1
		}

		go io.Copy(out, eout)
		go io.Copy(out, sout)
	}

	log.Debug("Exec starting")
	err := eCmd.Start()
	if err != nil {
		log.Error(err)
		out.Write([]byte(err.Error() + "\n\n"))
		return 1
	}

	log.Debug("Exec waiting")
	err = eCmd.Wait()
	if out != nil {
		out.Close()
	}

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

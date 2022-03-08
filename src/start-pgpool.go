package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	pgpool := run(false, "pgpool", "-n", "-f", "/app/vendor/pgpool/pgpool.conf", "-a", "/app/vendor/pgpool/pool_hba.conf")
	app := run(true, os.Args[1], os.Args[2:]...)

	go func() {
		pgpool.Wait()

		if app.Process != nil {
			app.Process.Kill()
		}
	}()

	app.Wait()

	if pgpool.Process != nil {
		pgpool.Process.Kill()
	}
}

func run(pipeStdin bool, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)

	if pipeStdin {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	return cmd
}

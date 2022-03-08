package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func main() {
	configure()

	pgpool := run(false, "pgpool", "-n", "-f", "/app/vendor/pgpool/pgpool.conf")
	app := run(true, os.Args[1], os.Args[2:]...)

	go func() {
		pgpool.Wait()

		if app.Process != nil {
			app.Process.Wait()
		}
	}()

	app.Wait()

	if pgpool.Process != nil {
		pgpool.Process.Wait()
	}
}

func configure() {
	configurePgpoolConf()
	configurePoolPasswd()
}

func configurePgpoolConf() {
	pgpoolConf, err := os.ReadFile("/app/vendor/pgpool/pgpool.conf.sample")

	if err != nil {
		log.Fatal(err)
	}

	pgpoolConf = append(pgpoolConf, `
		socket_dir = '/tmp'
		pcp_socket_dir = '/tmp'
		pool_passwd = '/app/vendor/pgpool/pool_passwd'
		ssl = on
		pid_file_name = '/tmp/pgpool.pid'
		logdir = '/tmp'
	`...)

	for i, postgresUrl := range postgresUrls() {
		host, port, _ := net.SplitHostPort(postgresUrl.Host)
		user := postgresUrl.User.Username()
		database := postgresUrl.Path[1:]

		if i == 0 {
			pgpoolConf = append(pgpoolConf, fmt.Sprintf(`
				sr_check_user = '%[1]s'
				sr_check_database = '%[2]s'
			
				health_check_user = '%[1]s'
				health_check_database = '%[2]s'
			`, user, database)...)
		}

		pgpoolConf = append(pgpoolConf, fmt.Sprintf(`
			backend_hostname%[1]d = '%[2]s'
			backend_port%[1]d = %[3]s
			backend_weight%[1]d = 1
			backend_data_directory%[1]d = '/data'
			backend_flag%[1]d = 'ALLOW_TO_FAILOVER'
		`, i, host, port)...)
	}

	err = os.WriteFile("/app/vendor/pgpool/pgpool.conf", pgpoolConf, 0600)

	if err != nil {
		log.Fatal(err)
	}
}

func configurePoolPasswd() {
	poolPasswd := ""

	for _, postgresUrl := range postgresUrls() {
		user := postgresUrl.User.Username()
		password, _ := postgresUrl.User.Password()
		poolPasswd += fmt.Sprintf("%s:md5%s\n", user, md5.Sum([]byte(password)))
	}

	err := os.WriteFile("/app/vendor/pgpool/pool_passwd", []byte(poolPasswd), 0600)

	if err != nil {
		log.Fatal(err)
	}
}

func postgresUrls() []*url.URL {
	pgpoolUrls := strings.Split(os.Getenv("PGPOOL_URLS"), " ")

	if len(pgpoolUrls) == 0 {
		log.Fatal("PGPOOL_URLS is not set")
	}

	postgresUrls := make([]*url.URL, len(pgpoolUrls))

	for i, pgpoolUrl := range pgpoolUrls {
		postgresUrl := os.Getenv(pgpoolUrl)

		if postgresUrl == "" {
			log.Fatal(pgpoolUrl + " is not set")
		}

		postgresUrlUrl, err := url.Parse(postgresUrl)
		if err != nil {
			log.Println(err)
			log.Fatal(pgpoolUrl + " is invalid")
		}

		postgresUrls[i] = postgresUrlUrl
	}

	return postgresUrls
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

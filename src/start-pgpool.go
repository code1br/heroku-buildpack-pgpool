package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	configure()

	var wg sync.WaitGroup
	sigterm := make(chan os.Signal, 1)

	signal.Ignore(syscall.SIGINT)
	signal.Notify(sigterm, syscall.SIGTERM)

	pgpool := run(true, "/app/.apt/usr/sbin/pgpool", "-n", "-f", "/app/vendor/pgpool/pgpool.conf")
	defer pgpool.Process.Kill()
	wg.Add(1)

	app := run(false, os.Args[1], os.Args[2:]...)
	defer app.Process.Kill()
	wg.Add(1)

	go func() {
		<-sigterm
		app.Process.Signal(syscall.SIGTERM)
	}()

	go func() {
		err := app.Wait()

		if err != nil {
			log.Println("app:", err)
		}

		if pgpool.Process != nil {
			pgpool.Process.Signal(syscall.SIGTERM)
		}

		wg.Done()
	}()

	go func() {
		err := pgpool.Wait()

		if err != nil {
			log.Println("pgpool:", err)
		}

		if app.Process != nil {
			app.Process.Signal(syscall.SIGTERM)
		}

		wg.Done()
	}()

	wg.Wait()
}

func configure() {
	configurePgpoolConf()
	configurePoolPasswd()
}

func configurePgpoolConf() {
	pgpoolConf, err := os.ReadFile("/app/.apt/usr/share/pgpool2/pgpool.conf")

	if err != nil {
		log.Fatal(err)
	}

	pgpoolConf = append(pgpoolConf, `
		socket_dir = '/tmp'
		pcp_socket_dir = '/tmp'
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
		poolPasswd += fmt.Sprintf("%s:md5%x\n", user, md5.Sum([]byte(password+user)))
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

func databaseUrl() string {
	postgresUrl := postgresUrls()[0]

	user := postgresUrl.User.Username()
	password, _ := postgresUrl.User.Password()
	database := postgresUrl.Path[1:]

	return fmt.Sprintf("postgres://%s:%s@localhost:9999/%s", user, password, database)
}

func run(pgpool bool, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)

	if pgpool {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	} else {
		cmd.Stdin = os.Stdin
		cmd.Env = append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", databaseUrl()))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	return cmd
}

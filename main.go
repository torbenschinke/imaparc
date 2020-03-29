package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.Server, "server", "", "the server to use")
	flag.StringVar(&cfg.Login, "login", "", "the login")
	flag.StringVar(&cfg.Password, "password", "", "password")
	flag.IntVar(&cfg.Port, "port", 993, "imap port")
	flag.BoolVar(&cfg.TLS, "tls", false, "use tls")
	flag.StringVar(&cfg.Dir, "dir", "", "the target directory to write the mails into")
	help := flag.Bool("help", false, "shows this help")

	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}

	app := &App{}
	err := app.Archive(cfg)
	if err != nil {
		fmt.Println("failed to archive:", err)
		os.Exit(1)
	}
	fmt.Println("archive completed")
}

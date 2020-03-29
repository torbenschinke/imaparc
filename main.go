package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.Server, "server", "", "the server to use")
	flag.StringVar(&cfg.Login, "login", "", "the login")
	flag.StringVar(&cfg.Password, "password", "", "password")
	flag.IntVar(&cfg.Port, "port", 993, "imap port")
	flag.BoolVar(&cfg.TLS, "tls", false, "use tls")
	flag.StringVar(&cfg.Dir, "dir", "", "the target directory to write the mails into")
	configFile := flag.String("configFile", "", "filename to a batch configuration in json format")
	help := flag.Bool("help", false, "shows this help")

	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}

	if len(*configFile) == 0 {
		singleMode(cfg)
	} else {
		batchMode(*configFile)
	}

	fmt.Println("archive completed")
}

func batchMode(cfgFile string) {
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		fmt.Printf("cannot read batch file %s: %v\n", cfgFile, err)
		os.Exit(2)
	}
	accounts := &AccountList{}
	err = json.Unmarshal(b, accounts)
	if err != nil {
		fmt.Printf("cannot decode json config %s: %v", cfgFile, err)
		os.Exit(3)
	}

	if accounts.Dir == "." {
		accounts.Dir = filepath.Dir(cfgFile)
	}

	for _, acc := range accounts.Accounts {
		cfg := &Config{Account: *acc}
		cfg.Dir = filepath.Join(accounts.Dir, acc.Name)
		singleMode(cfg)
	}
}

func singleMode(cfg *Config) {
	app := &App{}
	err := app.Archive(cfg)
	if err != nil {
		fmt.Println("failed to archive:", err)
		os.Exit(1)
	}
}

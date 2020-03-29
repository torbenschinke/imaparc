package main

type Config struct {
	Account
	Dir string
}

type Account struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Login    string `json:"login"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
}

type AccountList struct {
	Accounts []*Account `json:"accounts"`
	Dir      string     `json:"dir"`
}

package main

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"strconv"
)

type Imap struct {
	cfg         *Config
	client      *client.Client
	currentMbox string
}

func (i *Imap) Login(cfg *Config) error {
	i.cfg = cfg
	fmt.Printf("Connecting to server %s...\n", cfg.Server)

	// Connect to server
	if cfg.TLS {
		c, err := client.DialTLS(cfg.Server+":"+strconv.Itoa(cfg.Port), nil)
		if err != nil {
			return fmt.Errorf("failed to connect to tls server %s: %w", cfg.Server, err)
		}
		i.client = c
	} else {
		c, err := client.Dial(cfg.Server + ":" + strconv.Itoa(cfg.Port))
		if err != nil {
			return fmt.Errorf("failed to connect to server %s: %w", cfg.Server, err)
		}
		i.client = c
	}

	fmt.Println("Connected")

	// Login
	if err := i.client.Login(cfg.Login, cfg.Password); err != nil {
		return fmt.Errorf("username or password invalid: %w", err)
	}

	return nil
}

func (i *Imap) Logout() error {
	return i.client.Logout()
}

func (i *Imap) Mailboxes() ([]*imap.MailboxInfo, error) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- i.client.List("", "*", mailboxes)
	}()

	var res []*imap.MailboxInfo
	for m := range mailboxes {
		res = append(res, m)
	}
	if err := <-done; err != nil {
		return res, fmt.Errorf("failed to read mailboxes: %w", err)
	}
	return res, nil
}

func (i *Imap) Status(mailbox string) (*imap.MailboxStatus, error) {
	mbox, err := i.client.Select(mailbox, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox '%s': %w", mailbox, err)
	}
	return mbox, nil
}

func (i *Imap) Mail(mailbox string, num int) (*imap.Message, error) {

	res, err := i.Mails(mailbox, []imap.FetchItem{
		imap.FetchBody,
		imap.FetchBodyStructure,
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		imap.FetchRFC822,
		//imap.FetchRFC822Header,
		imap.FetchRFC822Size,
		//	imap.FetchRFC822Text,
		imap.FetchUid,
	}, num, num)
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("failed to get mail #%d: %w", num, err)
	}
	return res[0], nil
}

func (i *Imap) Mails(mailbox string, fetchItem []imap.FetchItem, from, to int) ([]*imap.Message, error) {
	if i.currentMbox != mailbox {
		// Select INBOX
		_, err := i.client.Select(mailbox, true)
		if err != nil {
			return nil, fmt.Errorf("failed to select mailbox '%s': %w", mailbox, err)
		}
		//fmt.Printf("Flags for %s: %s\n", mbox.Name, mbox.Flags)
		i.currentMbox = mailbox
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(uint32(from), uint32(to))

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- i.client.Fetch(seqset, fetchItem, messages)
	}()

	var res []*imap.Message
	for msg := range messages {
		res = append(res, msg)
	}

	if err := <-done; err != nil {
		return res, err
	}
	expected := (to - from) + 1
	if len(res) != expected {
		return res, fmt.Errorf("expected %d but got %d mails", expected, len(res))
	}
	return res, nil
}

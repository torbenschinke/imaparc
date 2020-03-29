package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	imap2 "github.com/emersion/go-imap"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Server   string
	Port     int
	Login    string
	Password string
	TLS      bool
	Dir      string
}

type App struct {
	cfg        *Config
	mailboxes  []*imap2.MailboxStatus
	totalMails int
}

func (a *App) Archive(cfg *Config) error {
	a.cfg = cfg
	imap := &Imap{}
	err := imap.Login(cfg)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	defer imap.Logout()

	mailboxes, err := imap.Mailboxes()
	if err != nil {
		return fmt.Errorf("unable to list mailboxes: %w", err)
	}
	a.mailboxes = nil
	a.totalMails = 0
	for _, mb := range mailboxes {
		fmt.Println(mb.Name)
		status, err := imap.Status(mb.Name)
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}
		fmt.Printf(" contains %d mails\n", status.Messages)
		a.totalMails += int(status.Messages)
		a.mailboxes = append(a.mailboxes, status)
	}
	fmt.Printf("total mails %d\n", a.totalMails)

	for _, mb := range a.mailboxes {
		err := a.saveMailbox(imap, mb)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) saveMailbox(srv *Imap, mailbox *imap2.MailboxStatus) error {
	targetDir := filepath.Join(a.cfg.Dir, sanitize(mailbox.Name))
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", targetDir, err)
	}
	if mailbox.Messages > 0 {
		mails, err := srv.Mails(mailbox.Name, []imap2.FetchItem{imap2.FetchEnvelope, imap2.FetchRFC822Size, imap2.FetchRFC822Header}, 1, int(mailbox.Messages))
		if err != nil {
			return fmt.Errorf("failed to fetch mails from %s: %w", mailbox.Name, err)
		}

		for _, mail := range mails {
			headers, err := bodyFor(mail, imap2.FetchRFC822Header)
			if err != nil {
				return fmt.Errorf("missing rfc header: %w", err)
			}
			hash := sha256.Sum224(headers)
			hashStr := hex.EncodeToString(hash[:])
			emlFile := filepath.Join(targetDir, hashStr+".eml")
			if _, err := os.Stat(emlFile); err != nil {
				fullMail, err := srv.Mail(mailbox.Name, int(mail.SeqNum)) //or uid?
				if err != nil {
					return fmt.Errorf("cannot read full mail: %w", err)
				}
				eml, err := bodyFor(fullMail, imap2.FetchRFC822)
				if err != nil {
					return fmt.Errorf("missing rfc content: %w", err)
				}
				err = ioutil.WriteFile(emlFile, eml, os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to write email %s: %w", emlFile, err)
				}
				fmt.Printf("saved %s/%d: %s\n", mailbox.Name, mail.SeqNum, debugTitle(mail))
			}

		}
	}
	return nil
}

func debugTitle(msg *imap2.Message) string {
	sb := &strings.Builder{}
	for _, adr := range msg.Envelope.From {
		sb.WriteString(adr.PersonalName)
		sb.WriteString("<")
		sb.WriteString(adr.MailboxName)
		sb.WriteString("@")
		sb.WriteString(adr.HostName)
		sb.WriteString(">")
	}
	sb.WriteString(" ")
	sb.WriteString(msg.Envelope.Subject)
	sb.WriteString(" ")
	sb.WriteString(msg.Envelope.Date.String())
	return sb.String()
}

func bodyFor(msg *imap2.Message, item imap2.FetchItem) ([]byte, error) {
	for k, v := range msg.Body {
		if k.FetchItem() == item {
			b, err := ioutil.ReadAll(v)
			if err != nil {
				return b, fmt.Errorf("failed to read body: %w", err)
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("cannot get body for %s", item)
}

func sanitize(str string) string {
	sb := &strings.Builder{}
	for _, r := range str {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

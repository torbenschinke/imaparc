# imap-archive
A small and simple command line tool to archive your imap mails in RFC822 format. Hashes the
RFC822 headers to detect changed or added emails before performing the download. Does never delete any
mails, just keeps adding.  

Supports also a batch mode, to archive a lot of servers in one step.

... and one more thing: there is also an integrated indexer and web ui.

## installation

```bash
# do not do this in a go module folder
go get -u github.com/torbenschinke/imaparc
go install github.com/torbenschinke/imaparc
```

## backup a single imap account

```bash
imaparc -server=mail.host.xy -port=993 -login=user -password=secret -tls=true -dir=/Users/user/mails
```

## backup batch

Create a json configuration like this:
```json
{
  "accounts": [
    {
      "name": "server-1-folder-name",
      "server": "mail.host1.xy",
      "port": 993,
      "login": "user1",
      "password": "secret1",
      "tls": true
    },
    {
      "name": "server-2-folder-name",
      "server": "mail.host2.xy",
      "port": 993,
      "login": "user2",
      "password": "secret2",
      "tls": true
    },
    {
      "name": "server-3-folder-name",
      "server": "mail.host3.xy",
      "port": 993,
      "login": "user3",
      "password": "secret3",
      "tls": true
    }
  ],
  "dir": "."
}
```

Finally invoke imaparc:
```bash
imaparc -configFile=/Users/home/mails/config.json
```

## Search engine
You can start an automatic indexer and web server to perform simple searches. Launch like this:

````bash
imaparc -searchDir=/Users/home/mails/mailarchive -searchHost=localhost -searchPort=8080
````

![Screenshot](example.png)
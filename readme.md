[![Build Status](https://travis-ci.com/nkonev/videochat.svg?branch=master)](https://travis-ci.com/nkonev/videochat)


# Architecture:

![Architecture](./.markdown/auth.png "Title")

## Add DNS names
`vim /etc/hosts`

```
127.0.0.1   api.site.local
```

## Allow container -> host

```bash
su -
firewall-cmd --permanent --zone=public --add-rich-rule='rule family=ipv4 source address="172.28.0.0/16" accept'
service firewalld restart
```

## Start docker-compose
```bash
docker-compose up -d
```

## Build static
Before development, you need to build static (html, sql). Please see `.travis.yml`

# Test in browser
Open `http://localhost:8081/chat` in Firefox main and an Anonymous window;
Login as `admin:admin` in main window and as `nikita:password` in the Anonymous window.
Create chat in main window and add `nikita` there.


## Generating password
```bash
sudo yum install -y httpd-tools

# generate password
htpasswd -bnBC 10 "" password | tr -d ':'
```


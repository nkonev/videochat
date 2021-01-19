[![Build Status](https://github.com/nkonev/videochat/workflows/CI%20jobs/badge.svg)](https://github.com/nkonev/videochat/actions)


# Architecture:

![Architecture](./.markdown/auth.png "Title")

## Add DNS names Mac OS
`vim /etc/hosts`

```
127.0.0.1   api.site.local
```

## Add DNS names Linux
`vim /etc/hosts`

```
127.0.0.1   api.site.local
127.0.0.1   host.docker.internal
```

## Allow container -> host (Linux)

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

<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>AAA OAuth2</title>

</head>
<body>
<div id="app-container">
    <#include "header.ftlh">
    <h1>OAuth2 login test</h1>

    <#list availableOauth2Providers as item>
        <a href="/api/aaa/login/oauth2/${item}" id="a-${item}">${item}</a> <br/>
    </#list>

</div>
</body>
</html>

<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Aaa OAuth2</title>

</head>
<body>
<div id="app-container">
    <#include "header.ftlh">
    <h1>OAuth2 login test</h1>

    <a href="/api/login/oauth2/facebook" id="a-facebook">Facebook</a>
    <a href="/api/login/oauth2/vkontakte" id="a-vkontakte">Vkontakte</a>
    <a href="/api/login/oauth2/google" id="a-google">Google</a>
    <a href="/api/login/oauth2/keycloak" id="a-keycloak">Keycloak</a>

</div>
</body>
</html>
<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>AAA Login</title>

</head>
<body>
<div id="app-container">
    <h1>Login page</h1>
    <form action="/api/aaa/login" method="post">
        <input id="login" name="login" type="text" placeholder="Login"/>
        <input id="password" name="password" type="password" placeholder="Password"/>
        <#if _csrf??>
            <input name="${_csrf.parameterName}" type="hidden" value="${_csrf.token}"/>
        </#if>
        <input id="btn-login" type="submit" value="Login"/>
    </form>

</div>
</body>
</html>

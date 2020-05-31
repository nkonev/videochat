<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Aaa Login</title>

</head>
<body>
<div id="app-container">
    <h1>Login page</h1>
    <#assign form=JspTaglibs["http://www.springframework.org/tags/form"]/>

    <@form.form action="/api/login" method="post">
        <input id="username" name="username" type="text" placeholder="Username"/>
        <input id="password" name="password" type="password" placeholder="Password"/>
        <input id="btn-login" type="submit" value="Login"/>
    </@form.form>

</div>
</body>
</html>
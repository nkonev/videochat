<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>AAA Error</title>
</head>

<body>
<div>
    <div>An error occurred</div>
    <#if timestamp??>
        <div>
            timestamp: ${timestamp?datetime}
        </div>
    </#if>
    <#if error??>
        <div>
            error: ${error}
        </div>
    </#if>
    <#if message??>
        <div>
            message: ${message}
        </div>
    </#if>
    <#if status??>
        <div>
            status: ${status}
        </div>
    </#if>
    <#if traceId??>
        <div>
            traceId: ${traceId}
        </div>
    </#if>
    <#if exception??>
        <div>
            exception: ${exception}
        </div>
    </#if>
    <#if trace??>
        <div>
            trace:
            <div style='white-space:pre-wrap;'>${trace}</div>
        </div>
    </#if>
</div>
</body>

</html>

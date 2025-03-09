<#if myPrincipal??>
    Logged as <b> ${myPrincipal.userLogin} </b>

    <div id="principalId">id: ${myPrincipal.identificator}</div>

    <#list myOauth2Identifiers as propName, propValue>
        ${propName}: ${(propValue)!"no id"} <br/>
    </#list>

    <form action="/api/aaa/logout" method="post">
        <input id="btn-logout" type="submit" value="Logout"/>
        <#if _csrf??>
            <input name="${_csrf.parameterName}" type="hidden" value="${_csrf.token}"/>
        </#if>
    </form>
</#if>

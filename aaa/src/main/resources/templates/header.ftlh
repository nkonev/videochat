<#assign sec=JspTaglibs["http://www.springframework.org/security/tags"]/>
<#assign form=JspTaglibs["http://www.springframework.org/tags/form"]/>

<@sec.authorize access="isAuthenticated()">
    Logged as <b> <@sec.authentication property="principal.username"/> </b>

    <div id="principalId">id: <@sec.authentication property="principal.id"/></div>

    <#list myOauth2Identifiers as propName, propValue>
        ${propName}: ${(propValue)!"no id"} <br/>
    </#list>

    <@form.form action="/api/aaa/logout" method="post">
        <input id="btn-logout" type="submit" value="Logout"/>
    </@form.form>
</@sec.authorize>

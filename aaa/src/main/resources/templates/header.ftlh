<#assign sec=JspTaglibs["http://www.springframework.org/security/tags"]/>
<#assign form=JspTaglibs["http://www.springframework.org/tags/form"]/>

<@sec.authorize access="isAuthenticated()">
    Logged as <b> <@sec.authentication property="principal.username"/> </b>

    <@form.form action="/api/logout" method="post">
        <input type="submit" value="Logout"/>
    </@form.form>
</@sec.authorize>

<#assign sec=JspTaglibs["http://www.springframework.org/security/tags"]/>
<#assign form=JspTaglibs["http://www.springframework.org/tags/form"]/>

<@sec.authorize access="isAuthenticated()">
    Logged as <b> <@sec.authentication property="principal.username"/> </b>

    <div>id: <@sec.authentication property="principal.id"/></div>
    <div>facebookId: <@sec.authentication property="principal.oauth2Identifiers.facebookId"/></div>
    <div>vkontakteId: <@sec.authentication property="principal.oauth2Identifiers.vkontakteId"/></div>
    <div>googleId: <@sec.authentication property="principal.oauth2Identifiers.googleId"/></div>

    <@form.form action="/api/logout" method="post">
        <input id="btn-logout" type="submit" value="Logout"/>
    </@form.form>
</@sec.authorize>

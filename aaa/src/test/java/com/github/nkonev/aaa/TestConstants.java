package com.github.nkonev.aaa;

public class TestConstants {
    public static final String USER = "${custom.it.user}";
    public static final String PASSWORD = "${custom.it.password}";
    public static final String USER_ID = "${custom.it.user.id}";

    public static final String USER_ALICE = "alice";
    public static final String USER_ALICE_PASSWORD = "password";
    public static final String USER_ADMIN = "admin";
    public static final String USER_BOB = "bob";

    public static final String USER_BOB_LDAP = "bobby";
    public static final String USER_BOB_LDAP_PASSWORD = "bobspassword"; // see in src/test/resources/test-server.ldif
    public static final String USER_BOB_LDAP_ID = "bobby"; // see in src/test/resources/test-server.ldif
    public static final String USER_NIKITA = "nikita";

    public static final String USER_LOCKED = "generated_user_66";
    public static final String COMMON_PASSWORD = "generated_user_password";
    public static final String COOKIE_XSRF = "VIDEOCHAT_XSRF_TOKEN";
    public static final String HEADER_XSRF_TOKEN = "X-XSRF-TOKEN";
    public static final String HEADER_COOKIE = "Cookie";
    public static final String HEADER_SET_COOKIE = "Set-Cookie";

    public static final String SQL_URL = "/sql";
    public static final String SQL_QUERY = "select * from fake_users;";
    public static final String USER_DETAILS = "/user-details-vuln";

    public static final String XSRF_TOKEN_VALUE = "xsrf";

    public static final String SESSION_COOKIE_NAME = "VIDEOCHAT_SESSION"; // see in src/test/resources/config/application.yml under server.servlet.session.cookie.name


}

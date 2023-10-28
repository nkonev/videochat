package com.github.nkonev.aaa;

/**
 * Created by nik on 28.05.17.
 */
public class TestConstants {

    public static final String SQL_URL = "/sql";
    public static final String SQL_QUERY = "select * from fake_users;";
    public static final String USER_DETAILS = "/user-details-vuln";

    public static final String USER_ALICE = CommonTestConstants.USER_ALICE;
    public static final String USER_ALICE_PASSWORD = CommonTestConstants.USER_ALICE_PASSWORD;
    public static final String USER_ADMIN  = CommonTestConstants.USER_ADMIN;
    public static final String USER_BOB = CommonTestConstants.USER_BOB;
    public static final String USER_NIKITA = CommonTestConstants.USER_NIKITA;

    public static final String ALLOW_IFRAME_SRC_STRING = "^(https://www\\.youtube\\.com.*)|(https://coub\\.com/.*)|(https://player\\.vimeo\\.com.*)$";
}

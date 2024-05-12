package name.nkonev.aaa;


import java.util.Set;

/**
 * Created by nik on 23.05.17.
 */
public class Constants {

    public static class Urls {
        public static final String ROOT = "/";
        public static final String PUBLIC_API = "/api/aaa";
        public static final String INTERNAL_API = "/internal";
        public static final String PROFILE = "/profile";
        public static final String SETTINGS = "/settings";
        public static final String SMILEYS = "/smileys";

        public static final String LANGUAGE = "/language";

        public static final String INIT = "/init";
        public static final String AUTH = "/auth";
        public static final String REGISTER = "/register";
        public static final String REGISTER_CONFIRM = "/register/confirm"; // html for handle link from email
        public static final String CHANGE_EMAIL_CONFIRM = "/change-email/confirm"; // html for handle link from email
        public static final String UUID = "uuid";
        public static final String RESEND_CONFIRMATION_EMAIL = "/resend-confirmation-email";

        public static final String RESEND_CHANGE_EMAIL_CONFIRM = "/change-email/resend";
        public static final String USER = "/user";

        public static final String ONLINE = "/online";
        public static final String LIST = "/list";

        public static final String SEARCH = "/search";

        public static final String LOCK = "/lock";

        public static final String CONFIRM = "/confirm";
        public static final String USER_ID = "/{"+PathVariables.USER_ID+"}";
        public static final String REQUEST_PASSWORD_RESET = "/request-password-reset";
        public static final String PASSWORD_RESET_SET_NEW = "/password-reset-set-new";
        public static final String SESSIONS = "/sessions";
        public static final String ROLE = "/role";
        public static final String REQUEST_FOR_ONLINE = "/request-for-online";
    }

    public static class Headers {
        public static final String X_AUTH_USERNAME = "X-Auth-Username";
        public static final String X_AUTH_USER_ID = "X-Auth-UserId";
        public static final String X_AUTH_EXPIRESIN = "X-Auth-ExpiresIn";
        public static final String X_AUTH_ROLE = "X-Auth-Role";
        public static final String X_AUTH_SESSION_ID = "X-Auth-SessionId";

        public static final String X_AUTH_AVATAR = "X-Auth-Avatar";
    }

    public static class PathVariables {
        public static final String USER_ID = "userId";
    }

    public static final String DELETED = "deleted";

    public static final Long NonExistentUser = -65000L;

    public static final Set<String> FORBIDDEN_USERNAMES = Set.of(DELETED, "all", "here");

    public static final int MIN_PASSWORD_LENGTH = 6;
    public static final int MAX_PASSWORD_LENGTH = 30;
    public static final int MAX_USERS_RESPONSE_LENGTH = 100;

    public static final int MAX_SMILEYS_LENGTH = 2048;

    public static final String DATE_FORMAT = "yyyy-MM-dd'T'HH:mm:ss.SSS'Z'";

}

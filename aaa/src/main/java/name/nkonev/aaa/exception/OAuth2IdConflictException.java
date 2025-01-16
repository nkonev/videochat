package name.nkonev.aaa.exception;


import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.OAuth2Error;

public class OAuth2IdConflictException extends OAuth2AuthenticationException {
    public OAuth2IdConflictException(String msg) {
        super(new OAuth2Error("merge_conflict"), msg);
    }
}

package name.nkonev.aaa.exception;


public class UserConflictException extends RuntimeException {
    public UserConflictException(String msg) {
        super(msg);
    }
}

package name.nkonev.aaa.exception;

public class ForbiddenActionException extends RuntimeException {

    private static final long serialVersionUID = 1885108427978294154L;

    public ForbiddenActionException(String message) {
        super(message);
    }

    public ForbiddenActionException() { }
}

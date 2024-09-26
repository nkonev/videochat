package name.nkonev.aaa.exception;

public class UnauthorizedException extends RuntimeException {

    private static final long serialVersionUID = 1885108427978294154L;

    public UnauthorizedException(String message) {
        super(message);
    }

    public UnauthorizedException() { }
}

package name.nkonev.aaa.exception;

public class DataNotFoundException extends RuntimeException {
    private static final long serialVersionUID = -7106664788237375370L;

    public DataNotFoundException(String message) {
        super(message);
    }

    public DataNotFoundException() { }
}

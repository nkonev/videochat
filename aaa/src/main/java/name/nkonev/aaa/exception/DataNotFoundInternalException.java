package name.nkonev.aaa.exception;

public class DataNotFoundInternalException extends RuntimeException {
    private static final long serialVersionUID = -7006664788237375370L;

    public DataNotFoundInternalException(String message) {
        super(message);
    }

    public DataNotFoundInternalException() { }
}

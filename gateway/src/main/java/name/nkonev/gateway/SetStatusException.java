package name.nkonev.gateway;

public class SetStatusException extends RuntimeException {
    private int status;
    public SetStatusException(String message, int status) {
        super(message);
        this.status = status;
    }

    public int getStatus() {
        return status;
    }

    @Override
    public String toString() {
        return "SetStatusException{" +
                "status=" + status +
                ", message=" + getMessage() +
                '}';
    }
}

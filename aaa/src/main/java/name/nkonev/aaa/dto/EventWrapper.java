package name.nkonev.aaa.dto;

public record EventWrapper<E>(
        E event,
        String type,
        boolean canThrottle
) {
    public EventWrapper<?> withCanThrottle(boolean b) {
        return new EventWrapper<>(this.event, this.type, b);
    }
}

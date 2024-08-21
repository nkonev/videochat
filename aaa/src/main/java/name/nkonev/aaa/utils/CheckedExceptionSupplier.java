package name.nkonev.aaa.utils;

@FunctionalInterface
public interface CheckedExceptionSupplier<T> {

    /**
     * Gets a result.
     *
     * @return a result
     */
    T get() throws Exception;
}

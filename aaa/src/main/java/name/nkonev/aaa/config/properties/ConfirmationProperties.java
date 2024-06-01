package name.nkonev.aaa.config.properties;

public record ConfirmationProperties(
    RegistrationProperties registration,
    ChangeEmailProperties changeEmail
) {
}

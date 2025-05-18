package name.nkonev.aaa.config.properties;

import java.util.HashSet;
import java.util.Set;

public record LoginProperties(
        boolean skipCharactersValidation,
        Set<String> additionalAllowedCharacters // additional to alphabetic and digits
) {

    public Set<String> getAdditionalAllowedCharacters() {
        return additionalAllowedCharacters != null ? additionalAllowedCharacters : new HashSet<>();
    }
}

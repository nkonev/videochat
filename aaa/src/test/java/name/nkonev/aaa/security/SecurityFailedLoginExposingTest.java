package name.nkonev.aaa.security;

import name.nkonev.aaa.AbstractTestRunner;
import org.junit.jupiter.api.Test;
import org.springframework.test.context.TestPropertySource;

import static name.nkonev.aaa.security.AaaUserDetailsService.MESSAGE_WITH_EXPOSED_SECRET;
import static org.assertj.core.api.Assertions.assertThat;


@TestPropertySource(locations = {"classpath:/config/security-fail-login.yml"})
public class SecurityFailedLoginExposingTest extends AbstractTestRunner {
    @Test
    public void testTheMessageIsNotShown() {
        var res = rawLoginDecodeError(username, password);
        assertThat(res.dto().getBody().message()).doesNotContain(MESSAGE_WITH_EXPOSED_SECRET);
        assertThat(res.dto().getBody().message()).isEqualTo("Unauthorized");
    }
}
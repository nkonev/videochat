package name.nkonev.aaa.services.tasks;

import com.nimbusds.jwt.JWTParser;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.entity.rest.KeycloakRoleEntity;
import name.nkonev.aaa.entity.rest.KeycloakUserEntity;
import name.nkonev.aaa.entity.rest.KeycloakUserInRoleEntity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.boot.autoconfigure.condition.ConditionalOnExpression;
import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientProperties;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.stereotype.Component;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.net.URI;
import java.text.ParseException;
import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

import static name.nkonev.aaa.security.OAuth2Providers.KEYCLOAK;
import static name.nkonev.aaa.utils.TimeUtil.convertToLocalDateTime;
import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

@ConditionalOnExpression("'${spring.security.oauth2.client.registration.keycloak.client-id:}' != ''")
@Component
public class KeycloakClient {

    private final RestTemplate restTemplate;

    private final AtomicReference<String> tokenHolder = new AtomicReference<>();

    private final Duration delta;

    private final String clientId;
    private final String clientSecret;

    private final String protocolHostPort;
    private final String realm;

    private static final Logger LOGGER = LoggerFactory.getLogger(KeycloakClient.class);

    public KeycloakClient(RestTemplate restTemplate, AaaProperties aaaProperties, ObjectProvider<OAuth2ClientProperties> oAuth2ClientProperties) {
        this.restTemplate = restTemplate;
        this.delta = aaaProperties.keycloak().tokenDelta();
        var o = oAuth2ClientProperties.getIfAvailable();
        if (o == null) {
            throw new IllegalStateException("OAuth2 should be enabled");
        }
        var kcp = o.getProvider().get(KEYCLOAK);
        if (kcp == null) {
            throw new IllegalStateException("You must define OAuth2 provider named 'keycloak'");
        }
        // http://localhost:8484/realms/my_realm2
        var issuerUrl = URI.create(kcp.getIssuerUri());

        var protocolHostPortBuilder = issuerUrl.getScheme();
        protocolHostPortBuilder += "://";
        protocolHostPortBuilder += issuerUrl.getHost();
        if (issuerUrl.getPort() != 0) {
            protocolHostPortBuilder += (":" + issuerUrl.getPort());
        }
        protocolHostPort = protocolHostPortBuilder;

        realm = issuerUrl.getPath().split("/")[2];

        var kcr = o.getRegistration().get(KEYCLOAK);
        if (kcr == null) {
            throw new IllegalStateException("You must define OAuth2 registration named 'keycloak'");
        }
        clientId = kcr.getClientId();
        clientSecret = kcr.getClientSecret();

        LOGGER.info("Configured with protocolHostPort = {}, realm = {}, clientId = {}", protocolHostPort, realm, clientId);
    }

    public List<KeycloakUserEntity> getUsers(int limit, int offset) {
        var token = getToken();
        var reqEntity = RequestEntity.get(protocolHostPort + "/admin/realms/" + realm + "/users?first=" + offset + "&max=" + limit)
                .accept(MediaType.APPLICATION_JSON)
                .header(HttpHeaders.AUTHORIZATION, "Bearer " + token).build();
        var respEntity = restTemplate.exchange(reqEntity, new ParameterizedTypeReference<List<KeycloakUserEntity>>() {});
        return respEntity.getBody();
    }

    public List<KeycloakUserInRoleEntity> getUsersInRole(String role, int limit, int offset) {
        var token = getToken();
        var reqEntity = RequestEntity.get(protocolHostPort + "/admin/realms/" + realm + "/roles/" + role + "/users?first=" + offset + "&max=" + limit)
                .accept(MediaType.APPLICATION_JSON)
                .header(HttpHeaders.AUTHORIZATION, "Bearer " + token).build();
        var respEntity = restTemplate.exchange(reqEntity, new ParameterizedTypeReference<List<KeycloakUserInRoleEntity>>() {});
        return respEntity.getBody();
    }

    public List<KeycloakRoleEntity> getRoles(int limit, int offset) {
        var token = getToken();
        var reqEntity = RequestEntity.get(protocolHostPort + "/admin/realms/" + realm + "/roles?first=" + offset + "&max=" + limit)
                .accept(MediaType.APPLICATION_JSON)
                .header(HttpHeaders.AUTHORIZATION, "Bearer " + token).build();
        var respEntity = restTemplate.exchange(reqEntity, new ParameterizedTypeReference<List<KeycloakRoleEntity>>() {});
        return respEntity.getBody();
    }

    private String getToken() {
        try {
            var token = tokenHolder.get();
            if (token != null) {
                if (checkNotExpired(token)) {
                    return token;
                }
            }

            MultiValueMap<String, String> urlParams = new LinkedMultiValueMap<>();
            urlParams.set("grant_type", "client_credentials");
            urlParams.set("client_id", clientId);
            urlParams.set("client_secret", clientSecret);
            var reqEntity = RequestEntity.post(protocolHostPort + "/realms/" + realm + "/protocol/openid-connect/token")
                    .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                    .body(urlParams);
            var respEntity = restTemplate.exchange(reqEntity, Map.class);
            var gotToken = respEntity.getBody().get("access_token").toString();
            tokenHolder.set(gotToken);
            LOGGER.info("Got new Keycloak client's token");
            return gotToken;
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    private boolean checkNotExpired(String token) throws ParseException {
        var parsed = JWTParser.parse(token);
        var exp = convertToLocalDateTime(parsed.getJWTClaimsSet().getExpirationTime());
        return exp.isAfter(getNowUTC().minus(delta));
    }
}

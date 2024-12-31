package name.nkonev.aaa.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import name.nkonev.aaa.dto.SuccessfulLoginDTO;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.core.Authentication;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationSuccessHandler;
import org.springframework.stereotype.Component;
import java.io.IOException;

@Component
public class RESTAuthenticationSuccessHandler extends SimpleUrlAuthenticationSuccessHandler {

    @Autowired
    private ObjectMapper objectMapper;

    @Override
    public void onAuthenticationSuccess(HttpServletRequest request, HttpServletResponse response,
                                        Authentication authentication) throws IOException, ServletException {

        clearAuthenticationAttributes(request);
        response.setContentType(MediaType.APPLICATION_JSON_UTF8_VALUE);

        Long id = ((UserAccountDetailsDTO)authentication.getPrincipal()).getId();

        SuccessfulLoginDTO successfulLoginDTO = new SuccessfulLoginDTO(id, "you successfully logged in");
        objectMapper.writeValue(response.getOutputStream(), successfulLoginDTO);
    }
}

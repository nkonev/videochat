package name.nkonev.users;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import java.util.Map;

@RestControllerAdvice
public class MyExceptionHandler {

    @ResponseBody
    @ResponseStatus(HttpStatus.I_AM_A_TEAPOT)
    @org.springframework.web.bind.annotation.ExceptionHandler(NullPointerException.class)
    public Map badRequest(NullPointerException e)  {
        return Map.of("I am", "not careful");
    }

}

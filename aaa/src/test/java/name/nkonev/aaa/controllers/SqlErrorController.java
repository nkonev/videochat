package name.nkonev.aaa.controllers;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.TestConstants;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class SqlErrorController {

    @GetMapping(Constants.Urls.EXTERNAL_API + TestConstants.SQL_URL)
    public void getSql() {
        throw new DataIntegrityViolationException(TestConstants.SQL_QUERY);
    }
}

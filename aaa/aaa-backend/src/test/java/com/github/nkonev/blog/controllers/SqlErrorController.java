package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import springfox.documentation.annotations.ApiIgnore;

@ApiIgnore
@RestController
public class SqlErrorController {

    @GetMapping(Constants.Urls.API + TestConstants.SQL_URL)
    public void getSql() {
        throw new DataIntegrityViolationException(TestConstants.SQL_QUERY);
    }
}

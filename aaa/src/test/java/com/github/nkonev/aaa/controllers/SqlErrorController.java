package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.TestConstants;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class SqlErrorController {

    @GetMapping(Constants.Urls.PUBLIC_API + TestConstants.SQL_URL)
    public void getSql() {
        throw new DataIntegrityViolationException(TestConstants.SQL_QUERY);
    }
}

package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.TestConstants;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserAccountDetailsDTOController {
    @GetMapping(Constants.Urls.API + TestConstants.USER_DETAILS)
    public UserAccountDetailsDTO getUserDetails() {
        return new UserAccountDetailsDTO();
    }

}

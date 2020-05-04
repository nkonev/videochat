package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import springfox.documentation.annotations.ApiIgnore;

@ApiIgnore
@RestController
public class UserAccountDetailsDTOController {
    @GetMapping(Constants.Urls.API + TestConstants.USER_DETAILS)
    public UserAccountDetailsDTO getUserDetails() {
        return new UserAccountDetailsDTO();
    }

}

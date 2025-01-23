package name.nkonev.aaa.controllers;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.TestConstants;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserAccountDetailsDTOController {
    @GetMapping(Constants.Urls.EXTERNAL_API + TestConstants.USER_DETAILS)
    public UserAccountDetailsDTO getUserDetails() {
        return new UserAccountDetailsDTO(null, null, null, null, null, false, false, true, true, null, null, false, null);
    }

}

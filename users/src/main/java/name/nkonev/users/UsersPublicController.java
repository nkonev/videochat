package name.nkonev.users;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UsersPublicController {
    public static class UserDto {
        private String name;

        public UserDto() {
        }

        public UserDto(String name) {
            this.name = name;
        }

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }
    }

    @Autowired
    private MyService myService;

    @GetMapping("/user")
    public UserDto get() {
        return new UserDto("Danny " + myService.getName());
    }
}

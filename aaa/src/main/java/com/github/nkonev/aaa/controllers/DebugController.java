package com.github.nkonev.aaa.controllers;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

@Controller
public class DebugController {

    @GetMapping({"/index.html", "/"})
    public String index() {
        return "index";
    }

    @GetMapping({"/oauth2.html"})
    public String oauth2() {
        return "oauth2";
    }

    @GetMapping({"/login.html"})
    public String login() {
        return "login";
    }

}

package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.services.RendertronFilter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;

import javax.servlet.http.HttpServletRequest;
import java.util.Optional;

@Controller
public class IndexController {

    @Autowired
    private Optional<RendertronFilter> rendertronFilter;

    @GetMapping({"/index.html", "/"})
    public String index(Model model, HttpServletRequest httpServletRequest) {

        rendertronFilter
                .filter(f1 -> f1.shouldRenderSeoScript(httpServletRequest))
                .ifPresent(f2 -> model.addAttribute("seoScript", f2.getSeoScript()));

        return "index";
    }
}

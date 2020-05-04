package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.config.ApplicationConfig;
import com.github.nkonev.blog.dto.ApplicationDTO;
import com.github.nkonev.blog.dto.BaseApplicationDTO;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.ArrayList;
import java.util.List;

import static com.github.nkonev.blog.Constants.Urls.API;
import static com.github.nkonev.blog.Constants.Urls.APPLICATION;

@RestController
public class ApplicationController {

    @Autowired
    private ApplicationConfig applicationConfig;

    @GetMapping(API+APPLICATION)
    public List<ApplicationDTO> getApplications(){
        List<ApplicationDTO> applicationDTOS = new ArrayList<>();
        if (applicationConfig.isEnableApplications()) {
            List<BaseApplicationDTO> applications = applicationConfig.getApplications();
            for (int i = 0; i < applications.size(); ++i) {
                BaseApplicationDTO baseApplicationDTO = applications.get(i);
                if (baseApplicationDTO.isEnabled()) {
                    applicationDTOS.add(new ApplicationDTO(i, baseApplicationDTO.getTitle(), baseApplicationDTO.getSrcUrl(), baseApplicationDTO.isEnabled()));
                }
            }
        }
        return applicationDTOS;
    }

    public boolean isEnableApplications() {
        return !getApplications().isEmpty();
    }
}

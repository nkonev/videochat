package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.dto.SettingsDTO;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.dto.UserRole;
import com.github.nkonev.blog.entity.jdbc.RuntimeSettings;
import com.github.nkonev.blog.repository.jdbc.RuntimeSettingsRepository;
import com.github.nkonev.blog.security.BlogSecurityService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.sql.SQLException;
import java.util.Arrays;
import java.util.stream.StreamSupport;

import static com.github.nkonev.blog.Constants.Urls.API;
import static com.github.nkonev.blog.Constants.Urls.CONFIG;

@RestController
public class SettingsController {

    @Autowired
    private RuntimeSettingsRepository runtimeSettingsRepository;

    @Autowired
    private BlogSecurityService blogSecurityService;

    @Autowired
    private ImageSettingsUploadController imageSettingsUploadController;

    @Autowired
    private ApplicationController applicationConfig;

    public static final String IMAGE_PART = "image";
    public static final String DTO_PART = "dto";


    public static final String IMAGE_BACKGROUND = "image.background";
    private static final String HEADER = "header";
    private static final String SUB_HEADER = "header.sub";
    private static final String TITLE_TEMPLATE = "title.template";
    private static final String BACKGROUND_COLOR = "background.color";

    private static final Logger LOGGER = LoggerFactory.getLogger(SettingsController.class);

    @GetMapping(API+CONFIG)
    public SettingsDTO getConfig(@AuthenticationPrincipal UserAccountDetailsDTO userAccount){

        Iterable<RuntimeSettings> runtimeSettings = runtimeSettingsRepository.findAll();

        SettingsDTO settingsDTOPartial = StreamSupport.stream(runtimeSettings.spliterator(), false)
                .reduce(
                        new SettingsDTO(),
                        (settingsDTO, runtimeSettings1) -> {
                            if (runtimeSettings1.getKey() == null) {
                                throw new RuntimeException("Null key is not supported");
                            }
                            switch (runtimeSettings1.getKey()) {
                                case IMAGE_BACKGROUND:
                                    settingsDTO.setImageBackground(runtimeSettings1.getValue());
                                    break;
                                case HEADER:
                                    settingsDTO.setHeader(runtimeSettings1.getValue());
                                    break;
                                case SUB_HEADER:
                                    settingsDTO.setSubHeader(runtimeSettings1.getValue());
                                    break;
                                case TITLE_TEMPLATE:
                                    settingsDTO.setTitleTemplate(runtimeSettings1.getValue());
                                    break;
                                case BACKGROUND_COLOR:
                                    settingsDTO.setBackgroundColor(runtimeSettings1.getValue());
                                    break;
                                default:
                                    LOGGER.warn("Unknown key " + runtimeSettings1.getKey());
                            }
                            return settingsDTO;
                        },
                        (settingsDTO, settingsDTO2) -> { throw new UnsupportedOperationException("Parallel is not supported");}
        );

        boolean canShowSettings = blogSecurityService.hasSettingsPermission(userAccount);

        settingsDTOPartial.setCanShowSettings(canShowSettings);
        settingsDTOPartial.setCanShowApplications(applicationConfig.isEnableApplications());
        settingsDTOPartial.setAvailableRoles(Arrays.asList(UserRole.ROLE_ADMIN, UserRole.ROLE_USER));

        return settingsDTOPartial;
    }

    @Transactional
    @PostMapping(value = API+CONFIG, consumes = {"multipart/form-data"})
    @PreAuthorize("@blogSecurityService.hasSettingsPermission(#userAccount)")
    public SettingsDTO putConfig(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestPart(value = DTO_PART) SettingsDTO dto,
            @RequestPart(value = IMAGE_PART, required = false) MultipartFile imagePart
    ) throws SQLException, IOException {
        var runtimeSettingsBackgroundImage = runtimeSettingsRepository.findById(IMAGE_BACKGROUND).orElseThrow();
        // update image
        if (imagePart != null && !imagePart.isEmpty()) {
            var imageResponse = imageSettingsUploadController.postImage(imagePart, userAccount);
            String relativeUrl = imageResponse.getRelativeUrl();
            runtimeSettingsBackgroundImage.setValue(relativeUrl);
        }
        // remove image
        if (Boolean.TRUE.equals(dto.getRemoveImageBackground())) {
            runtimeSettingsBackgroundImage.setValue(null);
        }
        runtimeSettingsBackgroundImage = runtimeSettingsRepository.save(runtimeSettingsBackgroundImage);

        // update header
        var runtimeSettingsHeader = runtimeSettingsRepository.findById(HEADER).orElseThrow();
        runtimeSettingsHeader.setValue(dto.getHeader());
        runtimeSettingsHeader = runtimeSettingsRepository.save(runtimeSettingsHeader);

        // update subheader
        var runtimeSettingsSubHeader = runtimeSettingsRepository.findById(SUB_HEADER).orElseThrow();
        runtimeSettingsSubHeader.setValue(dto.getSubHeader());
        runtimeSettingsSubHeader = runtimeSettingsRepository.save(runtimeSettingsSubHeader);

        // update title
        var runtimeTitleTemplate = runtimeSettingsRepository.findById(TITLE_TEMPLATE).orElseThrow();
        runtimeTitleTemplate.setValue(dto.getTitleTemplate());
        runtimeTitleTemplate = runtimeSettingsRepository.save(runtimeTitleTemplate);

        // background color
        var backgroundColor = runtimeSettingsRepository.findById(BACKGROUND_COLOR).orElseThrow();
        backgroundColor.setValue(dto.getBackgroundColor());
        backgroundColor = runtimeSettingsRepository.save(backgroundColor);

        return getConfig(userAccount);
    }
}

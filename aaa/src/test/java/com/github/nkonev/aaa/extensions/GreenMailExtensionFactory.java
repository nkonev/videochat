package com.github.nkonev.aaa.extensions;

import com.icegreen.greenmail.configuration.GreenMailConfiguration;
import com.icegreen.greenmail.util.ServerSetupTest;

public class GreenMailExtensionFactory {
    public static GreenMailExtension build() {
        return new GreenMailExtension(ServerSetupTest.SMTP_IMAP).withConfiguration(GreenMailConfiguration.aConfig().withDisabledAuthentication());
    }
}

package com.github.nkonev.blog.util;


import org.junit.jupiter.api.Assertions;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class UrlParser {
    private static final String regex = "\\(?\\b(http://|www[.])[-A-Za-z0-9+&@#/%?=~_()|!:,.;]*[-A-Za-z0-9+&@#/%=~_()|]";
    private static final Pattern p = Pattern.compile(regex);

    public static String parseUrlFromMessage(String message) {
        Matcher m = p.matcher(message);
        Assertions.assertTrue(m.find());
        return m.group();
    }
}

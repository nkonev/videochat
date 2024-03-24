package com.github.nkonev.aaa.utils;

import org.owasp.encoder.Encode;

public abstract class NullEncode {
    public static String forHtmlAttribute(String input) {
        if (input == null) {
            return input;
        }
        return Encode.forHtmlAttribute(input);
    }

    public static String forHtml(String input) {
        if (input == null) {
            return input;
        }
        return Encode.forHtml(input);
    }

}

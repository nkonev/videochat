package name.nkonev.aaa.utils;

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

    public static String forHtmlAndFixQuotes(String input) {
        var encoded = forHtml(input);
        if (encoded == null) {
            return encoded;
        }
        var t = encoded.replace("&#39;", "'");
        t = t.replace("&#34;", "\"");
        return t;
    }

}

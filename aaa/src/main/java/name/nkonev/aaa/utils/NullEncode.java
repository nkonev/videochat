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

    public static String forHtmlLogin(String input) {
        if (input == null) {
            return input;
        }
        var tinput = input.replace("'", "");
        tinput = tinput.replace("\"", "");
        tinput = tinput.replace("<", "");
        tinput = tinput.replace(">", "");

        var encoded = forHtml(tinput);
        if (encoded == null) {
            return encoded;
        }
        var t = encoded;
        return t;
    }

    public static String forHtmlEmail(String input) {
        return forHtmlLogin(input);
    }
}

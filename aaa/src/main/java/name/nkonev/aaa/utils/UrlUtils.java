package name.nkonev.aaa.utils;

import java.net.URI;
import java.util.List;

public abstract class UrlUtils {
    public static boolean containsUrl(List<String> elems, String elem) {
        try {
            var parsedUrlToTest = URI.create(elem);
            for (String s : elems) {
                var parsedAllowedUrl = URI.create(s);

                try {
                    if (nullEqual(parsedAllowedUrl.getHost(), parsedUrlToTest.getHost()) && parsedAllowedUrl.getPort() == parsedUrlToTest.getPort() && nullEqual(parsedAllowedUrl.getScheme(), parsedUrlToTest.getScheme())) {
                        return true;
                    }
                } catch (Exception ignore) { }
            }
            return false;
        } catch (Exception e) {
            return false;
        }
    }

    private static boolean nullEqual(String a, String b) {
        if (a == null && b == null) {
            return true;
        } else {
            return a.equals(b);
        }
    }
}

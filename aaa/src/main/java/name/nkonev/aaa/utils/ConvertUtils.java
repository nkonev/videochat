package name.nkonev.aaa.utils;

import org.springframework.util.StringUtils;

import javax.naming.NamingEnumeration;
import javax.naming.NamingException;
import java.util.HashSet;
import java.util.Set;

public abstract class ConvertUtils {
    public static boolean convertToBoolean(String value) {
        if (value == null) {
            return false;
        }
        value = value.trim();
        if (!StringUtils.hasLength(value)) {
            return false;
        }
        value = value.toLowerCase();
        return value.equals("true") || value.equals("yes") || value.equals("1");
    }

    public static Set<String> convertToStrings(NamingEnumeration rawRoles) {
        try {
            var res = new HashSet<String>();
            while (rawRoles.hasMore()) {
                res.add(NullUtils.getOrNullWrapException(() -> rawRoles.next().toString()));
            }
            return res;
        } catch (NamingException e) {
            throw new RuntimeException(e);
        }
    }

}

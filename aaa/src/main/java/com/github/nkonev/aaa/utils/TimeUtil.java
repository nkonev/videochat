package com.github.nkonev.aaa.utils;

import java.time.LocalDateTime;
import java.time.ZoneOffset;

/**
 * Created by nkonev on 15.05.17.
 */
public class TimeUtil {
    public static LocalDateTime getNowUTC() {
        return LocalDateTime.now(ZoneOffset.UTC);
    }
}

package com.github.nkonev.blog.utils;

import org.springframework.core.io.Resource;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.stream.Collectors;

public class ResourceUtils {
    public static String stringFromResource(Resource resource) {
        try(BufferedReader br = new BufferedReader(new InputStreamReader(resource.getInputStream()));) {
            return br.lines().collect(Collectors.joining("\n"));
        } catch (IOException e){
            throw new RuntimeException(e);
        }
    }

    public static String stringFromResourceOrNullIfNotExists(Resource resource) {
        if (resource != null && resource.exists()) {
            return stringFromResource(resource);
        } else {
            return null;
        }
    }
}

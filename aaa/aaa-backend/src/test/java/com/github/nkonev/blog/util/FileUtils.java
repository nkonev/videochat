package com.github.nkonev.blog.util;

import java.io.File;
import java.util.Arrays;

public class FileUtils {
    public static File getExistsFile(String... fileCandidates) {
        for(String fileCandidate: fileCandidates) {
            File file = new File(fileCandidate);
            if (file.exists()) {
                return file;
            }
        }
        throw new RuntimeException("exists file not found among " + Arrays.toString(fileCandidates));
    }
}

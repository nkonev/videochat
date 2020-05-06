package com.github.nkonev.blog.utils;

/**
 * Created by nik on 05.07.17.
 */
public class PageUtils {
    public static final int MAX_SIZE=100;
    public static final int DEFAULT_SIZE=20;

    public static int fixPage(int page){
        return page < 0 ? 0 : page;
    }

    public static int fixSize(int size){
        return (size > MAX_SIZE || size<1) ? DEFAULT_SIZE : size;
    }

    public static int getOffset(int page, int size) {
        return page * size;
    }
}

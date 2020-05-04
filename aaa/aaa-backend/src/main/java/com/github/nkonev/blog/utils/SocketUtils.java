package com.github.nkonev.blog.utils;

import java.io.IOException;
import java.net.Socket;

public class SocketUtils {
    public static boolean isTcpPortFree(String host, int port) {
        try (Socket ignored = new Socket(host, port)) {
            return false;
        } catch (IOException ignored) {
            return true;
        }
    }
}

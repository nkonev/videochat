package com.github.nkonev.blog.services.port;

import com.github.nkonev.blog.utils.SocketUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.TimeUnit;

public abstract class AbstractPortChecker {
    abstract protected Logger getLogger();

    protected void check(final int maxCount, final String host, final int port){
        int i = 0;
        while (SocketUtils.isTcpPortFree(host, port) && i<maxCount){
            ++i;
            try {
                getLogger().info("{}/{} host {} port {} is not available for connection", i, maxCount, host, port);
                TimeUnit.SECONDS.sleep(1);
            } catch (Exception e){
                getLogger().warn("Error during check host "+host+" port "+port, e);
            }
        }
        if (i == maxCount) {
            throw new RuntimeException("Max count=" + maxCount +" for check host "+host+" port "+port +" is reached");
        }
    }
}

package utils;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.Arrays;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.stream.Collectors;

public class ProcessUtils {
    private static final Logger LOGGER = LoggerFactory.getLogger(ProcessUtils.class);

    public static class ProcessInfo {
        public String stdout;
        public String stderr;
        public int exitCode;

        public ProcessInfo(String stdout, String stderr, int exitCode) {
            this.stdout = stdout;
            this.stderr = stderr;
            this.exitCode = exitCode;
        }
    }

    private static final String STDERR = "stderr";
    private static final String STDOUT = "stdout";

    private static String[] split(String s) {
        return s.split("\\s+");
    }

    public static Process launch(String line) throws IOException {
        return launch(line, processBuilder -> {}, true);
    }

    /**
     * Launch process and write its stdout to log
     * @param line
     * @param builderCustomize
     * @return
     * @throws IOException
     */
    public static Process launch(String line, Consumer<ProcessBuilder> builderCustomize) throws IOException {
        return launch(line, builderCustomize, true);
    }

    public static Process launch(String... cmd) throws IOException {
        return launch(null, true, cmd);
    }

    /**
     *
     * @param line
     * @param builderCustomize
     * @param inheritIo consume IO and write it to SLF4J
     * @return
     * @throws IOException
     */
    public static Process launch(String line, Consumer<ProcessBuilder> builderCustomize, boolean inheritIo) throws IOException {
        final String[] splitted = split(line);
        return launch(builderCustomize, inheritIo, splitted);
    }

    /**
     *
     * @param cmd
     * @param builderCustomize
     * @param inheritIo consume IO and write it to SLF4J
     * @return
     * @throws IOException
     */
    public static Process launch(Consumer<ProcessBuilder> builderCustomize, boolean inheritIo, String... cmd) throws IOException {
        final ProcessBuilder processBuilder = new ProcessBuilder();
        LOGGER.debug("Will run {}", Arrays.toString(cmd));
        processBuilder.command(cmd);
        if (builderCustomize != null) {
            builderCustomize.accept(processBuilder);
        }
        final Process p = processBuilder.start();
        if (inheritIo) {
            readStreamInDaemonAndClose(p.getInputStream(), p, STDOUT);
            readStreamInDaemonAndClose(p.getErrorStream(), p, STDERR);
        }
        return p;
    }

    private static String readAndClose(final InputStream is) throws IOException {
        final String s;
        try(BufferedReader bufferedReader = new BufferedReader(new InputStreamReader(is))) {
            s = bufferedReader.lines().collect(Collectors.joining("\n"));
        }
        return s;
    }

    /**
     * Run process and wait while it stops. Returns process stdout, stderror, exitcode.
     * @param logs
     * @return
     */
    public static ProcessInfo get(Process logs){
        LOGGER.debug("Started cycling logger process");

        try (final InputStream is = logs.getInputStream();
             final InputStream es = logs.getErrorStream();
        ) {
            int ec = logs.waitFor();
            String stdout = readAndClose(is);
            String stderr = readAndClose(es);
            return new ProcessInfo(stdout, stderr, ec);
        } catch (IOException e) {
            LOGGER.error("Error on get logs", e);
            throw new RuntimeException(e);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException(e);
        }
    }


    /**
     * Read stream in daemon and write to SLF4J. Daemon thread will close stream.
     * @param is
     * @param logs
     * @param name
     */
    private static void readStreamInDaemonAndClose(InputStream is, Process logs, String name) {
        final Logger LOGGER = LoggerFactory.getLogger(name);
        Thread stdOutThread = new Thread(() -> {
            try (BufferedReader stdoutReader = new BufferedReader(new InputStreamReader(is))) {
                while (!Thread.currentThread().isInterrupted() && logs.isAlive()) {
                    String s = stdoutReader.readLine();
                    if (s!=null) {
                        LOGGER.info(s);
                    }
                }
                LOGGER.debug("logger is interrupted");
            } catch (IOException e) {
                LOGGER.error("I/O Error in reader", e);
            } finally {
                try {
                    is.close();
                } catch (IOException e) {
                    LOGGER.error("error on closing stream", e);
                }
            }
        });
        stdOutThread.setDaemon(true);
        stdOutThread.start();
    }

    /**
     * Run dedicated daemon thread which logs.
     * @param command command whitespace separated, if cycle==true => "cat /path/to/log", false => "tail -f /path/to/log"
     * @param loggerName
     * @param cycle
     */
    public static void startLoggerThread(final String command, final String loggerName, final boolean cycle) {
        final Logger LOGGER = LoggerFactory.getLogger(loggerName);
        final int processRecreateIntervalSeconds = 1;

        if (cycle) {
            final Thread t = new Thread(() -> {
                while (!Thread.currentThread().isInterrupted()) {
                    try {
                        final ProcessBuilder processBuilder = new ProcessBuilder();
                        processBuilder.command(split(command));
                        LOGGER.debug("Starting cycling logger process");
                        final Process logs = processBuilder.start();
                        ProcessInfo processInfo = get(logs);
                        LOGGER.info("stdout:: \n" + processInfo.stdout);
                        LOGGER.info("stderr:: \n" + processInfo.stderr);

                        try {
                            TimeUnit.SECONDS.sleep(processRecreateIntervalSeconds);
                        } catch (InterruptedException e) {
                            Thread.currentThread().interrupt();
                        }
                    } catch (IOException e) {
                        LOGGER.error("Error on get logs", e);
                    }
                }
            });
            t.setDaemon(true);
            t.start();
        } else {
            try {
                final ProcessBuilder processBuilder = new ProcessBuilder();
                processBuilder.command(split(command));
                LOGGER.debug("Starting long logger process");
                final Process logs = processBuilder.start();
                LOGGER.debug("Started long logger process");

                final InputStream is = logs.getInputStream();
                final InputStream es = logs.getErrorStream();

                readStreamInDaemonAndClose(es, logs, STDERR);
                readStreamInDaemonAndClose(is, logs, STDOUT);

            } catch (IOException e) {
                LOGGER.error("Error on get logs", e);
            }
        }

    }

}

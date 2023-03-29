#/bin/bash
# You need to start EmulatorServersController firstly
# export JAVA_HOME=/usr/lib/jvm/bellsoft-java17.x86_64
./mvnw clean package -DskipTests
exec $JAVA_HOME/bin/java -jar target/aaa-0.0.0-exec.jar --spring.config.location=file:src/main/resources/config/application.yml,file:src/test/resources/config/oauth2-basic.yml

package name.nkonev.aaa.services.tasks;

import javax.naming.NamingException;
import javax.naming.directory.Attributes;

interface AttributesConsumer {
    void consumeFromAttributes(Attributes attributes) throws NamingException;
}

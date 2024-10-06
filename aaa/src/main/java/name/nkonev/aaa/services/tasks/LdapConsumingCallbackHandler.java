package name.nkonev.aaa.services.tasks;

import org.springframework.ldap.core.AttributesMapper;
import org.springframework.ldap.core.NameClassPairCallbackHandler;
import org.springframework.ldap.support.LdapUtils;

import javax.naming.NameClassPair;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.directory.SearchResult;

public class LdapConsumingCallbackHandler implements NameClassPairCallbackHandler {

    private final AttributesConsumer consumer;

    /**
     * Constructs a new instance around the specified {@link AttributesMapper}.
     *
     * @param consumer the target mapper.
     */
    public LdapConsumingCallbackHandler(AttributesConsumer consumer) {
        this.consumer = consumer;
    }

    /**
     * Cast the NameClassPair to a SearchResult and pass its attributes to the
     * {@link AttributesMapper}.
     *
     * @param nameClassPair a <code> SearchResult</code> instance.
     * @return the Object returned from the mapper.
     */
    public void getObjectFromNameClassPairInternal(NameClassPair nameClassPair) {
        if (!(nameClassPair instanceof SearchResult)) {
            throw new IllegalArgumentException("Parameter must be an instance of SearchResult");
        }

        SearchResult searchResult = (SearchResult) nameClassPair;
        Attributes attributes = searchResult.getAttributes();
        try {
            this.consumer.consumeFromAttributes(attributes);
            return;
        } catch (javax.naming.NamingException ex) {
            throw LdapUtils.convertLdapException(ex);
        }
    }

    @Override
    public final void handleNameClassPair(NameClassPair nameClassPair) throws NamingException {
        getObjectFromNameClassPairInternal(nameClassPair);
    }

}

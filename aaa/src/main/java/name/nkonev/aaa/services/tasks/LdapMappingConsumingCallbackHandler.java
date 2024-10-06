package name.nkonev.aaa.services.tasks;

import org.springframework.ldap.core.AttributesMapper;
import org.springframework.ldap.core.NameClassPairCallbackHandler;
import org.springframework.ldap.support.LdapUtils;

import javax.naming.NameClassPair;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.directory.SearchResult;
import java.util.ArrayList;
import java.util.List;
import java.util.function.Consumer;

public class LdapMappingConsumingCallbackHandler<T> implements NameClassPairCallbackHandler {

    private final AttributesMapper<T> mapper;

    private final Consumer<List<T>> consumer;

    private final List<T> list = new ArrayList<>();

    private final int batchSize;

    /**
     * Constructs a new instance around the specified {@link AttributesMapper}.
     *
     * @param mapper the target mapper.
     */
    public LdapMappingConsumingCallbackHandler(AttributesMapper<T> mapper, Consumer<List<T>> consumer, int batchSize) {
        this.mapper = mapper;
        this.consumer = consumer;
        this.batchSize = batchSize;
    }

    /**
     * Cast the NameClassPair to a SearchResult and pass its attributes to the
     * {@link AttributesMapper}.
     *
     * @param nameClassPair a <code> SearchResult</code> instance.
     * @return the Object returned from the mapper.
     */
    public T getObjectFromNameClassPairInternal(NameClassPair nameClassPair) {
        if (!(nameClassPair instanceof SearchResult)) {
            throw new IllegalArgumentException("Parameter must be an instance of SearchResult");
        }

        SearchResult searchResult = (SearchResult) nameClassPair;
        Attributes attributes = searchResult.getAttributes();
        try {
            return this.mapper.mapFromAttributes(attributes);
        } catch (javax.naming.NamingException ex) {
            throw LdapUtils.convertLdapException(ex);
        }
    }

    @Override
    public final void handleNameClassPair(NameClassPair nameClassPair) throws NamingException {
        this.list.add(getObjectFromNameClassPairInternal(nameClassPair));
        if (list.size() == batchSize) {
            this.consumer.accept(list);
            list.clear();
        }
    }

    public void processLeftovers() {
        if (!list.isEmpty()) {
            this.consumer.accept(list);
            list.clear();
        }
    }
}

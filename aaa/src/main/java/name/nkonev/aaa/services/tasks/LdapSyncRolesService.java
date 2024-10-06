package name.nkonev.aaa.services.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.RoleMapEntry;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.ldap.LdapUserInRoleEntity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ldap.core.LdapOperations;
import org.springframework.ldap.core.SearchExecutor;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.directory.DirContext;
import javax.naming.directory.SearchControls;
import java.util.ArrayList;
import java.util.List;
import java.util.function.Consumer;

import static name.nkonev.aaa.utils.ConvertUtils.extractExtId;

@Service
public class LdapSyncRolesService {

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private LdapOperations ldapOperations;

    private static final Logger LOGGER = LoggerFactory.getLogger(LdapSyncRolesService.class);

    public String getNecessaryAdminRole() {
        var list = aaaProperties.roleMappings().ldap().stream()
                .filter(roleMapEntry -> UserRole.ROLE_ADMIN.name().equals(roleMapEntry.our()))
                .map(RoleMapEntry::their)
                .toList();
        if (list.isEmpty()) {
            throw new IllegalStateException("Admin role not found in mapping");
        }
        return list.getFirst();
    }

    public void processRoles(int batchSize, String extRole, Consumer<List<LdapUserInRoleEntity>> batchProcessor) {
        var groupBase = aaaProperties.ldap().group().base();
        var groupName = aaaProperties.ldap().group().filter();

        var lq = LdapQueryBuilder.query().base(groupBase).filter(groupName, extRole);
        // partial copy-paste from LdapTemplate because of near Long.MAX_VALUE length of array in spliterator in Spliterators.spliteratorUnknownSize()
        SearchControls controls = new SearchControls();
        controls.setSearchScope(SearchControls.ONELEVEL_SCOPE);
        if (lq.searchScope() != null) {
            controls.setSearchScope(lq.searchScope().getId());
        }
        controls.setReturningObjFlag(true);
        SearchExecutor se = (DirContext ctx) -> {
            var filterValue = lq.filter().encode();
            LOGGER.debug("Executing search with base [{}] and filter [{}]", lq.base(), filterValue);
            return ctx.search(lq.base(), filterValue, controls);
        };
        var handler = new LdapConsumingCallbackHandler(a -> mapAttributesToInRoleEntity(batchSize, a, batchProcessor));
        ldapOperations.search(se, handler);
    }

    private void mapAttributesToInRoleEntity(int batchSize, Attributes attributes, Consumer<List<LdapUserInRoleEntity>> batchProcessor) throws NamingException {
        final List<LdapUserInRoleEntity> list = new ArrayList<>();

        javax.naming.NamingEnumeration<?> iter = null;
        try {
            iter = attributes.get(aaaProperties.ldap().attributeNames().role()).getAll();
            while (iter.hasMore()) {
                var extIdInRole = iter.next();
                var extId = extractExtId(aaaProperties.ldap().attributeNames(), extIdInRole);
                if (StringUtils.hasLength(extId)) {
                    list.add(new LdapUserInRoleEntity(extId));
                }
                if (list.size() == batchSize) {
                    batchProcessor.accept(list);
                    list.clear();
                }
            }
            // process leftovers
            if (!list.isEmpty()) {
                batchProcessor.accept(list);
                list.clear();
            }
        } catch (NamingException e) {
            throw new RuntimeException(e);
        } finally {
            if (iter != null) {
                iter.close();
            }
        }
    }

}

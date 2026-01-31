import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faFacebook } from '@fortawesome/free-brands-svg-icons/faFacebook'
import { faVk } from '@fortawesome/free-brands-svg-icons/faVk'
import { faGoogle } from '@fortawesome/free-brands-svg-icons/faGoogle'
import { faKey } from '@fortawesome/free-solid-svg-icons/faKey'
import { faDatabase } from '@fortawesome/free-solid-svg-icons/faDatabase'
import { faObjectGroup } from '@fortawesome/free-solid-svg-icons/faObjectGroup'

library.add(faFacebook, faVk, faGoogle, faKey, faDatabase, faObjectGroup);

export default FontAwesomeIcon;

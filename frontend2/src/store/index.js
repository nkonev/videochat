// Utilities
import {createPinia, setActivePinia, setMapStoreSuffix} from 'pinia';

const pinia = createPinia();

setActivePinia(pinia);

setMapStoreSuffix('');

export default pinia;

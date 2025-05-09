import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi' // Use the Material Design Icons (mdi)
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';

const vuetify = createVuetify({
    components,
    directives,
    icons: {
        defaultSet: 'mdi', // Set the default icon pack to mdi
        aliases,
        sets: { mdi },
  },
})

export default vuetify
